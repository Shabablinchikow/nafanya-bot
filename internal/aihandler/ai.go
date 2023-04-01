package aihandler

import (
	"context"
	"github.com/getsentry/sentry-go"
	"github.com/sashabaranov/go-openai"
	"log"
)

type Handler struct {
	ai *openai.Client
}

func NewHandler(ai *openai.Client) *Handler {
	return &Handler{
		ai: ai,
	}
}

func (h *Handler) GetPromptResponse(prompt string, userInput string) (string, error) {
	messages := make([]openai.ChatCompletionMessage, 0)
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: prompt,
	})

	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: userInput,
	})

	resp, err := h.ai.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    openai.GPT3Dot5Turbo,
			Messages: messages,
		})
	if err != nil {
		sentry.CaptureException(err)
		log.Println("Completion error:", err)
		return "", err
	}

	return resp.Choices[0].Message.Content, nil
}