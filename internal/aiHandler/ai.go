package aiHandler

import (
	"context"
	"github.com/sashabaranov/go-openai"
	"log"
	"math/rand"
	"time"
)

type Handler struct {
	ai *openai.Client
}

// emotionList is a list of strings containing all available emotions
var emotionList = []string{
	"с нейтральным отношением",
	"с пессимизмом",
	"с оптимизмом",
	"с сарказмом",
	"с раздражением",
	"с жестким негативом",
}

func NewHandler(ai *openai.Client) *Handler {
	return &Handler{
		ai: ai,
	}
}

func (h *Handler) GetQuestionResponse(prompt string) (string, error) {
	messages := make([]openai.ChatCompletionMessage, 0)
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: `Ты бот-помощник для чата "Кухня", где собравшиеся люди обсуждают политику без цензуры и опасений быть репрессированными. Твоя задача — отвечать на вопросы участников чата` + rollEmotion() + `, можно материться, но не более 140 слов. Следующим сообщением будет идти вопрос от участника чата. Начинай ответ с новой строки.`,
	})

	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: prompt,
	})

	resp, err := h.ai.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    openai.GPT3Dot5Turbo,
			Messages: messages,
		})
	if err != nil {
		log.Println("Completion error:", err)
		return "", err
	}

	return resp.Choices[0].Message.Content, nil
}

func (h *Handler) GetInterferenceResponse(prompt string) (string, error) {
	messages := make([]openai.ChatCompletionMessage, 0)
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: `Тебя зовут Нафаня. Ты учачтник чата "Кухня", где собравшиеся люди обсуждают политику без цензуры и опасений быть репрессированными. Твоя задача – вклиниваться в диалог участников чата ` + rollEmotion() + `, по возможности нужно материться, но не более 80 слов. Следующим сообщением будет идти сообщение от участника чата. Начинай ответ с новой строки.`,
	})

	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: prompt,
	})

	resp, err := h.ai.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    openai.GPT3Dot5Turbo,
			Messages: messages,
		})
	if err != nil {
		log.Println("Completion error:", err)
		return "", err
	}

	return resp.Choices[0].Message.Content, nil
}

func rollEmotion() string {
	s := rand.NewSource(time.Now().Unix())
	r := rand.New(s) // initialize local pseudorandom generator

	return emotionList[r.Intn(len(emotionList))]
}
