package aihandler

import (
	"cloud.google.com/go/vertexai/genai"
	"context"
	"fmt"
	"github.com/getsentry/sentry-go"
	"github.com/sashabaranov/go-openai"
	"log"
)

type Handler struct {
	aiOAI    *openai.Client
	aiGoogle *genai.Client
}

func NewHandler(oai *openai.Client, googleAI *genai.Client) *Handler {
	return &Handler{
		aiOAI:    oai,
		aiGoogle: googleAI,
	}
}

func (h *Handler) GetPromptResponse(prompt string, userInput string, model string, maxTokens int) (string, error) {
	switch model {
	case "oai":
		return h.GetPromptResponseOAI(prompt, userInput, maxTokens)
	case "google":
		return h.GetPromptResponseGoogle(prompt, userInput, maxTokens)
	}

	return "", fmt.Errorf("unknown model: %s", model)
}

func (h *Handler) GetPromptResponseOAI(prompt string, userInput string, maxTokens int) (string, error) {
	messages := make([]openai.ChatCompletionMessage, 0)
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: prompt,
	})

	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: userInput,
	})

	resp, err := h.aiOAI.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:     openai.GPT4o,
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
	model := h.aiGoogle.GenerativeModel("gemini-1.5-pro-preview-0514")

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
