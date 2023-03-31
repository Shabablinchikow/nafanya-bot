package tghandler

import (
	"encoding/json"
	"github.com/getsentry/sentry-go"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/shabablinchikow/nafanya-bot/internal/aihandler"
	"github.com/shabablinchikow/nafanya-bot/internal/domain"
	"log"
	"strconv"
	"strings"
	"time"
)

type Handler struct {
	bot    *tgbotapi.BotAPI
	ai     *aihandler.Handler
	db     *domain.Handler
	chats  []domain.Chat
	config domain.BotConfig
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
	config, err2 := db.GetBotConfig()
	if err2 != nil {
		panic(err2)
	}

	return &Handler{
		bot:    bot,
		ai:     ai,
		db:     db,
		chats:  channels,
		config: config,
	}
}

// HandleEvents handles the events from the bot API
func (h *Handler) HandleEvents(update tgbotapi.Update) {
	if h.checkChatExists(update.Message.Chat) {
		if h.checkAllowed(update.Message.Chat.ID) || h.isAdmin(update.Message.From.ID) {
			if update.Message != nil { // If we got a message
				log.Println(update.Message.IsCommand())
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

		}
	} else {
		channel := domain.GetDefaultChat()

		channel.ID = update.Message.Chat.ID
		channel.Type = update.Message.Chat.Type
		channel.ChatName = update.Message.Chat.Title

		err := h.db.CreateChannelConfig(channel)
		if err != nil {
			sentry.CaptureException(err)
			log.Println(err)
		}

		h.reloadChannels()
	}
}

func (h *Handler) commandHandler(update tgbotapi.Update) {
	log.Println(update.Message.Command())
	switch update.Message.Command() {
	case "start":
		h.startMessage(update)
	case "listChats":
		h.listChats(update)
	case "chat":
		h.chat(update)
	case "chatAddDays":
		h.chatAddDays(update)
	case "chatMakeVIP":
		h.chatMakeVIP(update)
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

func (h *Handler) listChats(update tgbotapi.Update) {
	if h.isAdmin(update.Message.From.ID) {
		var message string
		for _, chat := range h.chats {
			message += "/chat " + strconv.FormatInt(chat.ID, 10) + "\n Chat: " + chat.ChatName + "\n Type: " + chat.Type + "\n\n"
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, message)
		msg.ReplyToMessageID = update.Message.MessageID

		_, err := h.bot.Send(msg)
		if err != nil {
			log.Println(err)
		}
	}
}

func (h *Handler) chat(update tgbotapi.Update) {
	if h.isAdmin(update.Message.From.ID) {
		id, err := strconv.ParseInt(update.Message.CommandArguments(), 10, 64)
		if err != nil {
			log.Println(err)
			return
		}
		chat, err2 := h.db.GetChannelConfig(id)
		if err2 != nil {
			log.Println(err2)
			return
		}

		chatData, err3 := json.Marshal(chat)
		if err3 != nil {
			log.Println(err3)
			return
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, string(chatData))
		msg.ReplyToMessageID = update.Message.MessageID

		_, err4 := h.bot.Send(msg)
		if err4 != nil {
			log.Println(err4)
		}
	}
}

func (h *Handler) chatAddDays(update tgbotapi.Update) {
	if h.isAdmin(update.Message.From.ID) {
		args := strings.Split(update.Message.CommandArguments(), " ")
		if len(args) == 2 {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				log.Println(err)
				return
			}
			chat, err2 := h.db.GetChannelConfig(id)
			if err2 != nil {
				log.Println(err2)
				return
			}

			days, err3 := strconv.Atoi(args[1])
			if err3 != nil {
				log.Println(err3)
				return
			}

			chat.BilledTo = chat.BilledTo.AddDate(0, 0, days)
			err4 := h.db.UpdateChannelConfig(chat)
			if err4 != nil {
				log.Println(err4)
				return
			}

			h.reloadChannels()

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Done")
			msg.ReplyToMessageID = update.Message.MessageID

			_, err5 := h.bot.Send(msg)
			if err5 != nil {
				log.Println(err5)
			}
		} else {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Wrong arguments")
			msg.ReplyToMessageID = update.Message.MessageID

			_, err := h.bot.Send(msg)
			if err != nil {
				log.Println(err)
			}
		}

	}
}

func (h *Handler) chatMakeVIP(update tgbotapi.Update) {
	if h.isAdmin(update.Message.From.ID) {
		id, err := strconv.ParseInt(update.Message.CommandArguments(), 10, 64)
		if err != nil {
			log.Println(err)
			return
		}
		chat, err2 := h.db.GetChannelConfig(id)
		if err2 != nil {
			log.Println(err2)
			return
		}

		chat.BilledTo = time.Date(2077, 1, 1, 0, 0, 0, 0, time.UTC)
		err3 := h.db.UpdateChannelConfig(chat)
		if err3 != nil {
			log.Println(err3)
			return
		}

		h.reloadChannels()

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Done")
		msg.ReplyToMessageID = update.Message.MessageID

		_, err5 := h.bot.Send(msg)
		if err5 != nil {
			log.Println(err5)
		}
	}
}
