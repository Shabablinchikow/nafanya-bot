package aihandler

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/vertexai/genai"
	"github.com/getsentry/sentry-go"
	"github.com/sashabaranov/go-openai"
)

type Handler struct {
	aiOAI    *openai.Client
	deepSeek *openai.Client
	aiGoogle *genai.Client
}

func NewHandler(oai *openai.Client, googleAI *genai.Client, deep *openai.Client) *Handler {
	return &Handler{
		aiOAI:    oai,
		aiGoogle: googleAI,
		deepSeek: deep,
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
	return h.GetPromptResponseOAICommon(h.aiOAI, prompt, userInput, maxTokens, openai.GPT4o)
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

	return resp.Choices[0].Message.Content, nil
}
func (h *Handler) GetPromptResponseGoogle(prompt string, userInput string, maxTokens int) (string, error) {
	model := h.aiGoogle.GenerativeModel("gemini-2.0-flash-001")

	// Enable Google Search grounding
	searchTool := &genai.Tool{
		GoogleSearchRetrieval: &genai.GoogleSearchRetrieval{},
	}
	model.Tools = []*genai.Tool{searchTool}
	log.Printf("Model tools configured: %+v\n", model.Tools)
	if len(model.Tools) > 0 && model.Tools[0].GoogleSearchRetrieval != nil {
		log.Println("GoogleSearchRetrieval tool is configured.")
	}

	var safetySettings []*genai.SafetySetting
	safetySettings = append(safetySettings, &genai.SafetySetting{
		Category:  genai.HarmCategoryHarassment,
		Threshold: genai.HarmBlockOnlyHigh,
	})
	safetySettings = append(safetySettings, &genai.SafetySetting{
		Category:  genai.HarmCategoryHateSpeech,
		Threshold: genai.HarmBlockOnlyHigh,
	})
	safetySettings = append(safetySettings, &genai.SafetySetting{
		Category:  genai.HarmCategorySexuallyExplicit,
		Threshold: genai.HarmBlockOnlyHigh,
	})
	safetySettings = append(safetySettings, &genai.SafetySetting{
		Category:  genai.HarmCategoryDangerousContent,
		Threshold: genai.HarmBlockOnlyHigh,
	})

	model.SafetySettings = safetySettings
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(prompt)},
	}

	model.SetMaxOutputTokens(int32(maxTokens))
	model.SetCandidateCount(1)

	resp, err := model.GenerateContent(context.Background(), genai.Text(userInput))
	if err != nil {
		log.Println("Error generating content:", err)
		for _, cand := range resp.Candidates {
			reason := cand.FinishReason
			reasonString := reason.String()

			log.Println("reason: " + reasonString)

			for _, safe := range cand.SafetyRatings {
				log.Println("safety: " + safe.Category.String() + safe.Probability.String())
			}
		}
		return "Error generating answer: " + err.Error(), err
	}

	var respText string

	for _, cand := range resp.Candidates {
		log.Printf("Candidate FinishReason: %s\n", cand.FinishReason.String())
		if cand.CitationMetadata != nil {
			log.Println("CitationMetadata found:")
			if len(cand.CitationMetadata.CitationSources) > 0 {
				for _, source := range cand.CitationMetadata.CitationSources {
					log.Printf("  Citation Source URI: %s, StartIndex: %d, EndIndex: %d, License: %s\n", source.URI, source.StartIndex, source.EndIndex, source.License)
				}
			} else {
				log.Println("  No citation sources found in metadata.")
			}
		} else {
			log.Println("No CitationMetadata found for candidate.")
		}

		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				respText = fmt.Sprintf("%s%s", respText, part)
			}
		}
	}

	return respText, nil
}

func (h *Handler) GetImageFromPrompt(prompt string) (string, error) {
	img, err := h.aiOAI.CreateImage(context.Background(),
		openai.ImageRequest{
			Prompt:         prompt,
			N:              1,
			Size:           openai.CreateImageSize1792x1024,
			ResponseFormat: openai.CreateImageResponseFormatURL,
			Quality:        openai.CreateImageQualityHD,
			Model:          openai.CreateImageModelDallE3,
		})
	if err != nil {
		sentry.CaptureException(err)
		log.Println("Image error:", err)
		return "", err
	}

	return img.Data[0].URL, nil
}
