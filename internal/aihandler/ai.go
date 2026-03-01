package aihandler

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/vertexai/genai"
	"github.com/getsentry/sentry-go"
	"github.com/sashabaranov/go-openai"
	genaisdk "google.golang.org/genai"
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
	case "oai":
		return h.GetPromptResponseOAI(prompt, userInput, maxTokens)
	case "deepseek":
		return h.GetPromptResponseDS(prompt, userInput, maxTokens)
	case "google":
		return h.GetPromptResponseGoogle(prompt, userInput, maxTokens)
	}

	return "", fmt.Errorf("unknown model: %s", model)
}

func (h *Handler) GetPromptResponseOAI(prompt string, userInput string, maxTokens int) (string, error) {
	return h.GetPromptResponseOAICommon(h.aiOAI, prompt, userInput, maxTokens, "gpt-5")
}

func (h *Handler) GetPromptResponseDS(prompt string, userInput string, maxTokens int) (string, error) {
	return h.GetPromptResponseOAICommon(h.deepSeek, prompt, userInput, maxTokens, "deepseek-chat")
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

func (h *Handler) getPromptResponseGeminiDirect(prompt string, userInput string, maxTokens int) (string, error) {
	resp, err := h.geminiDirect.Models.GenerateContent(
		context.Background(),
		"gemini-3.1-pro-preview",
		genaisdk.Text(userInput),
		&genaisdk.GenerateContentConfig{
			SystemInstruction: genaisdk.NewContentFromText(prompt, "user"),
			MaxOutputTokens:   int32(maxTokens),
			SafetySettings: []*genaisdk.SafetySetting{
				{Category: genaisdk.HarmCategoryHarassment, Threshold: genaisdk.HarmBlockThresholdBlockOnlyHigh},
				{Category: genaisdk.HarmCategoryHateSpeech, Threshold: genaisdk.HarmBlockThresholdBlockOnlyHigh},
				{Category: genaisdk.HarmCategorySexuallyExplicit, Threshold: genaisdk.HarmBlockThresholdBlockOnlyHigh},
				{Category: genaisdk.HarmCategoryDangerousContent, Threshold: genaisdk.HarmBlockThresholdBlockOnlyHigh},
			},
		},
	)
	if err != nil {
		log.Println("Gemini direct error:", err)
		return "Error generating answer: " + err.Error(), err
	}
	return resp.Text(), nil
}

func (h *Handler) getPromptResponseVertexAI(prompt string, userInput string, maxTokens int) (string, error) {
	model := h.aiGoogle.GenerativeModel("gemini-2.0-flash-001")

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

	resp, err := model.GenerateContent(context.Background(), genai.Text(userInput))
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
		"imagen-4.0-fast-generate-001",
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

func (h *Handler) GetImageFromPrompt(prompt string) (string, error) {
	img, err := h.aiOAI.CreateImage(context.Background(),
		openai.ImageRequest{
			Prompt:         prompt,
			N:              1,
			Size:           openai.CreateImageSize1792x1024,
			ResponseFormat: openai.CreateImageResponseFormatURL,
			Quality:        openai.CreateImageQualityHD,
			Model:          "gpt-image-1.5",
		})
	if err != nil {
		sentry.CaptureException(err)
		log.Println("Image error:", err)
		return "", err
	}

	if len(img.Data) == 0 {
		err := fmt.Errorf("no image data returned from API")
		sentry.CaptureException(err)
		return "", err
	}

	return img.Data[0].URL, nil
}
