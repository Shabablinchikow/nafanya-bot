package tgHandler

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/shabablinchikow/nafanya-bot/internal/aiHandler"
	"log"
)

type Handler struct {
	bot *tgbotapi.BotAPI
	ai  *aiHandler.Handler
}

func NewHandler(bot *tgbotapi.BotAPI, ai *aiHandler.Handler) *Handler {
	return &Handler{
		bot: bot,
		ai:  ai,
	}
}

// HandleEvents handles the events from the bot API
func (h *Handler) HandleEvents(update tgbotapi.Update) {
	if checkAllowed(update.Message.Chat.ID) {
		if update.Message != nil { // If we got a message
			switch {
			case update.Message.IsCommand():
				h.commandHandler(update)
				break
			case isPersonal(update):
				h.personalHandler(update)
			case isItTime(update.Message.Chat.ID):
				h.randomInterference(update)
				break
			}
		}
	}
}

func (h *Handler) commandHandler(update tgbotapi.Update) {
	switch update.Message.Command() {
	case "start":
		h.startMessage(update)

	case "question":
		h.questionMessage(update)
	}
}

func (h *Handler) startMessage(update tgbotapi.Update) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Hello, I'm Nafanya Bot!")
	msg.ReplyToMessageID = update.Message.MessageID

	_, err := h.bot.Send(msg)
	if err != nil {
		log.Println(err)
	}
}

func (h *Handler) questionMessage(update tgbotapi.Update) {
	var message string
	ans, err := h.ai.GetQuestionResponse(update.Message.CommandArguments())
	if err != nil {
		log.Println(err)
		message = "Something went wrong"
	} else {
		message = ans
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, message)
	msg.ReplyToMessageID = update.Message.MessageID

	_, err2 := h.bot.Send(msg)
	if err2 != nil {
		log.Println(err2)
	}
}

func (h *Handler) randomInterference(update tgbotapi.Update) {
	var message string
	ans, err := h.ai.GetQuestionResponse(update.Message.CommandArguments())
	if err != nil {
		log.Println(err)
		message = "Something went wrong"
	} else {
		message = ans
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, message)
	msg.ReplyToMessageID = update.Message.MessageID

	_, err2 := h.bot.Send(msg)
	if err2 != nil {
		log.Println(err2)
	}
}

func (h *Handler) personalHandler(update tgbotapi.Update) {
	var message string
	ans, err := h.ai.GetQuestionResponse(update.Message.Text)
	if err != nil {
		log.Println(err)
		message = "Something went wrong"
	} else {
		message = ans
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, message)
	msg.ReplyToMessageID = update.Message.MessageID

	_, err2 := h.bot.Send(msg)
	if err2 != nil {
		log.Println(err2)
	}
}
