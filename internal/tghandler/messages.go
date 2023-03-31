package tghandler

import (
	"encoding/json"
	"github.com/getsentry/sentry-go"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/shabablinchikow/nafanya-bot/internal/aihandler"
	"github.com/shabablinchikow/nafanya-bot/internal/domain"
	"log"
	"strconv"
)

type Handler struct {
	bot      *tgbotapi.BotAPI
	ai       *aihandler.Handler
	db       *domain.Handler
	channels []domain.Channel
}

const (
	openAIErrorMessage = "Something went wrong with OpenAI API"
	helloMessage       = "Hello, I'm Nafanya Bot!"
)

func NewHandler(bot *tgbotapi.BotAPI, ai *aihandler.Handler, db *domain.Handler) *Handler {
	channels, err := db.GetAllChannelsConfig()
	if err != nil {
		sentry.CaptureException(err)
		panic(err)
	}

	return &Handler{
		bot:      bot,
		ai:       ai,
		db:       db,
		channels: channels,
	}
}

// HandleEvents handles the events from the bot API
func (h *Handler) HandleEvents(update tgbotapi.Update) {
	if h.checkAllowed(update.Message.Chat.ID) {
		if update.Message != nil { // If we got a message
			switch {
			case update.Message.IsCommand():
				h.commandHandler(update)
			case isPersonal(update):
				h.personalHandler(update)
			case isItTime(update.Message.Chat.ID):
				h.randomInterference(update)
			}
		}
	} else {
		chatID, err := h.bot.GetChat(tgbotapi.ChatInfoConfig{ChatConfig: tgbotapi.ChatConfig{ChatID: update.Message.Chat.ID}})
		if err != nil {
			sentry.CaptureException(err)
			log.Println(err)
		}
		rawChatData, _ := json.Marshal(chatID)
		message := "Can't process message from chat " + chatID.Title + "with ID " + strconv.FormatInt(chatID.ID, 10) + "and raw data \n" + string(rawChatData)
		msg := tgbotapi.NewMessage(438663, message)

		_, err2 := h.bot.Send(msg)
		if err2 != nil {
			sentry.CaptureException(err2)
			log.Println(err2)
		}
	}
}

func (h *Handler) commandHandler(update tgbotapi.Update) {
	switch update.Message.Command() {
	case "start":
		h.startMessage(update)
	}
}

func (h *Handler) startMessage(update tgbotapi.Update) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, helloMessage)
	msg.ReplyToMessageID = update.Message.MessageID

	_, err := h.bot.Send(msg)
	if err != nil {
		sentry.CaptureException(err)
		log.Println(err)
	}
}

func (h *Handler) randomInterference(update tgbotapi.Update) {
	var message string
	ans, err := h.ai.GetPromptResponse(h.promptCompiler(update.Message.Chat.ID, RandomInterference, update))
	if err != nil {
		sentry.CaptureException(err)
		log.Println(err)
		message = openAIErrorMessage
	} else {
		message = ans
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, message)
	msg.ReplyToMessageID = update.Message.MessageID

	_, err2 := h.bot.Send(msg)
	if err2 != nil {
		sentry.CaptureException(err2)
		log.Println(err2)
	}
}

func (h *Handler) personalHandler(update tgbotapi.Update) {
	var message string
	ans, err := h.ai.GetPromptResponse(h.promptCompiler(update.Message.Chat.ID, Question, update))
	if err != nil {
		sentry.CaptureException(err)
		log.Println(err)
		message = openAIErrorMessage
	} else {
		message = ans
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, message)
	msg.ReplyToMessageID = update.Message.MessageID

	_, err2 := h.bot.Send(msg)
	if err2 != nil {
		sentry.CaptureException(err2)
		log.Println(err2)
	}
}
