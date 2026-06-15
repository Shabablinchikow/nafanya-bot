package aihandler

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"strings"

	"cloud.google.com/go/vertexai/genai"
	"github.com/getsentry/sentry-go"
	"github.com/shabablinchikow/nafanya-bot/internal/cfg"
	"github.com/sashabaranov/go-openai"
	genaisdk "google.golang.org/genai"
	"mvdan.cc/xurls/v2"
)

type Handler struct {
	aiOAI        *openai.Client
	deepSeek     *openai.Client
	aiGoogle     *genai.Client
	geminiDirect *genaisdk.Client
}

func NewHandler(oai *openai.Client, googleAI *genai.Client, deep *openai.Client, geminiDirect *genaisdk.Client) *Handler {
	return &Handler{
		aiOAI:        oai,
		aiGoogle:     googleAI,
		deepSeek:     deep,
		geminiDirect: geminiDirect,
	}
}

func (h *Handler) GetPromptResponse(prompt string, userInput string, model string, maxTokens int) (string, error) {
	switch model {
	case string(cfg.AIModelGPT55):
		return h.GetPromptResponseOAI(prompt, userInput, maxTokens)
	case string(cfg.AIModelDeepSeekV4):
		return h.GetPromptResponseDS(prompt, userInput, maxTokens)
	case string(cfg.AIModelGemini35):
		return h.GetPromptResponseGoogle(prompt, userInput, maxTokens)
	}

	return "", fmt.Errorf("unknown model: %s", model)
}

func (h *Handler) GetPromptResponseOAI(prompt string, userInput string, maxTokens int) (string, error) {
	return h.GetPromptResponseOAICommon(h.aiOAI, prompt, userInput, maxTokens, cfg.GetAIModelBackendName(cfg.AIModelGPT55))
}

func (h *Handler) GetPromptResponseDS(prompt string, userInput string, maxTokens int) (string, error) {
	return h.GetPromptResponseOAICommon(h.deepSeek, prompt, userInput, maxTokens, cfg.GetAIModelBackendName(cfg.AIModelDeepSeekV4))
}

func (h *Handler) GetPromptResponseOAICommon(client *openai.Client, prompt string, userInput string, maxTokens int, model string) (string, error) {
	if client == nil {
		return "", fmt.Errorf("model not available")
	}
	messages := make([]openai.ChatCompletionMessage, 0)
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: prompt,
	})

	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: userInput,
	})

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:     model,
			Messages:  messages,
			MaxTokens: maxTokens,
		})
	if err != nil {
		sentry.CaptureException(err)
		log.Println("Completion error:", err)
		return "", err
	}

	if len(resp.Choices) == 0 {
		err := fmt.Errorf("no choices returned from API")
		sentry.CaptureException(err)
		return "", err
	}

	return resp.Choices[0].Message.Content, nil
}
func (h *Handler) GetPromptResponseGoogle(prompt string, userInput string, maxTokens int) (string, error) {
	if h.geminiDirect != nil {
		return h.getPromptResponseGeminiDirect(prompt, userInput, maxTokens)
	}
	return h.getPromptResponseVertexAI(prompt, userInput, maxTokens)
}

const geminiRetries = 3

// extractYouTubeURLs finds YouTube URLs in userInput, returns them and the cleaned text.
func extractYouTubeURLs(userInput string) ([]string, string) {
	rxRelaxed := xurls.Relaxed()
	allURLs := rxRelaxed.FindAllString(userInput, -1)
	remaining := userInput
	var ytURLs []string
	for _, u := range allURLs {
		if strings.Contains(u, "youtube.com") || strings.Contains(u, "youtu.be") {
			ytURLs = append(ytURLs, u)
			remaining = strings.ReplaceAll(remaining, u, "")
		}
	}
	return ytURLs, strings.TrimSpace(remaining)
}

func (h *Handler) getPromptResponseGeminiDirect(prompt string, userInput string, _ int) (string, error) {

	geminiCfg := &genaisdk.GenerateContentConfig{
		SystemInstruction: genaisdk.NewContentFromText(prompt, "user"),
		SafetySettings: []*genaisdk.SafetySetting{
			{Category: genaisdk.HarmCategoryHarassment, Threshold: genaisdk.HarmBlockThresholdBlockOnlyHigh},
			{Category: genaisdk.HarmCategoryHateSpeech, Threshold: genaisdk.HarmBlockThresholdBlockOnlyHigh},
			{Category: genaisdk.HarmCategorySexuallyExplicit, Threshold: genaisdk.HarmBlockThresholdBlockOnlyHigh},
			{Category: genaisdk.HarmCategoryDangerousContent, Threshold: genaisdk.HarmBlockThresholdBlockOnlyHigh},
		},
	}

	ytURLs, cleanedInput := extractYouTubeURLs(userInput)
	parts := make([]*genaisdk.Part, 0, len(ytURLs)+1)
	for _, u := range ytURLs {
		parts = append(parts, genaisdk.NewPartFromURI(u, "video/mp4"))
	}
	parts = append(parts, genaisdk.NewPartFromText(cleanedInput))
	contents := []*genaisdk.Content{genaisdk.NewContentFromParts(parts, "user")}

	var lastErr error
	for i := range geminiRetries {
		resp, err := h.geminiDirect.Models.GenerateContent(
			context.Background(),
			cfg.GetAIModelBackendName(cfg.AIModelGemini35),
			contents,
			geminiCfg,
		)
		if err != nil {
			log.Printf("Gemini attempt %d error: %v", i+1, err)
			lastErr = err
			continue
		}
		return resp.Text(), nil
	}
	return "", lastErr
}


func (h *Handler) getPromptResponseVertexAI(prompt string, userInput string, maxTokens int) (string, error) {
	model := h.aiGoogle.GenerativeModel(cfg.VertexAIModel())

	model.SafetySettings = []*genai.SafetySetting{
		{Category: genai.HarmCategoryHarassment, Threshold: genai.HarmBlockOnlyHigh},
		{Category: genai.HarmCategoryHateSpeech, Threshold: genai.HarmBlockOnlyHigh},
		{Category: genai.HarmCategorySexuallyExplicit, Threshold: genai.HarmBlockOnlyHigh},
		{Category: genai.HarmCategoryDangerousContent, Threshold: genai.HarmBlockOnlyHigh},
	}
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(prompt)},
	}
	model.SetMaxOutputTokens(int32(maxTokens))
	model.SetCandidateCount(1)

	ytURLs, cleanedInput := extractYouTubeURLs(userInput)
	vParts := make([]genai.Part, 0, len(ytURLs)+1)
	for _, u := range ytURLs {
		vParts = append(vParts, genai.FileData{FileURI: u, MIMEType: "video/mp4"})
	}
	vParts = append(vParts, genai.Text(cleanedInput))
	resp, err := model.GenerateContent(context.Background(), vParts...)
	if err != nil {
		log.Println("Error generating content:", err)
		return "Error generating answer: " + err.Error(), err
	}

	var respText string
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				respText = fmt.Sprintf("%s%s", respText, part)
			}
		}
	}
	return respText, nil
}

func (h *Handler) GetImageFromPromptBanana(prompt string) ([]byte, string, error) {
	if h.geminiDirect == nil {
		return nil, "", fmt.Errorf("banana unavailable: GEMINI_DIRECT_KEY not configured")
	}

	resp, err := h.geminiDirect.Models.GenerateImages(
		context.Background(),
		cfg.GetImageModelBackendName(cfg.ImageModelGemini31),
		prompt,
		&genaisdk.GenerateImagesConfig{
			NumberOfImages: 1,
			OutputMIMEType: "image/jpeg",
		},
	)
	if err != nil {
		log.Println("Banana image error:", err)
		return nil, "", err
	}

	if len(resp.GeneratedImages) == 0 || resp.GeneratedImages[0].Image == nil {
		return nil, "", fmt.Errorf("no image data returned from Imagen")
	}

	img := resp.GeneratedImages[0].Image
	return img.ImageBytes, img.MIMEType, nil
}

func (h *Handler) GetImageFromPrompt(prompt string) ([]byte, string, error) {
	// ponytail: gpt-image-* rejects response_format and always returns b64_json
	img, err := h.aiOAI.CreateImage(context.Background(),
		openai.ImageRequest{
			Prompt:  prompt,
			N:       1,
			Size:    openai.CreateImageSize1536x1024,
			Quality: openai.CreateImageQualityHigh,
			Model:   cfg.GetImageModelBackendName(cfg.ImageModelGPTImage2),
		})
	if err != nil {
		sentry.CaptureException(err)
		log.Println("Image error:", err)
		return nil, "", err
	}

	if len(img.Data) == 0 || img.Data[0].B64JSON == "" {
		err := fmt.Errorf("no image data returned from API")
		sentry.CaptureException(err)
		return nil, "", err
	}

	data, err := base64.StdEncoding.DecodeString(img.Data[0].B64JSON)
	if err != nil {
		sentry.CaptureException(err)
		return nil, "", err
	}

	return data, "image/png", nil
}
