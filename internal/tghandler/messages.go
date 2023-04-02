package tghandler

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/getsentry/sentry-go"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/shabablinchikow/nafanya-bot/internal/aihandler"
	"github.com/shabablinchikow/nafanya-bot/internal/domain"
	"golang.org/x/exp/slices"
	"log"
	"strconv"
	"strings"
	"time"
)

type Handler struct {
	bot       *tgbotapi.BotAPI
	ai        *aihandler.Handler
	db        *domain.Handler
	chats     []domain.Chat
	config    domain.BotConfig
	chatCache map[int64]chatCache
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
		bot:       bot,
		ai:        ai,
		db:        db,
		chats:     channels,
		config:    config,
		chatCache: make(map[int64]chatCache),
	}
}

// HandleEvents handles the events from the bot API
func (h *Handler) HandleEvents(update tgbotapi.Update) {
	defer sentry.Recover()
	if update.Message != nil { // If we got a message
		if h.checkChatExists(update.Message.Chat) {
			ctx := context.WithValue(context.Background(), "chat", update.Message.Chat.ID)
			switch {
			case update.Message.IsCommand():
				span := sentry.StartSpan(ctx, "command", sentry.TransactionName("Handle tg command"))
				h.commandHandler(update)
				span.Finish()
			case h.isPersonal(update):
				span := sentry.StartSpan(ctx, "personal", sentry.TransactionName("Handle tg personal message"))
				h.personalHandler(update)
				span.Finish()
			case h.isItTime(update.Message.Chat.ID):
				span := sentry.StartSpan(ctx, "random", sentry.TransactionName("Handle tg random interference"))
				h.randomInterference(update)
				span.Finish()
			}
		} else {
			channel := domain.GetDefaultChat()

			channel.ID = update.Message.Chat.ID
			channel.Type = update.Message.Chat.Type
			if update.Message.Chat.Type == "private" {
				channel.ChatName = update.Message.Chat.FirstName + " " + update.Message.Chat.LastName
			} else {
				channel.ChatName = update.Message.Chat.Title
			}

			err := h.db.CreateChannelConfig(channel)
			if err != nil {
				sentry.CaptureException(err)
				log.Println(err)
			}

			h.reloadChannels()
		}
	}
}

func (h *Handler) commandHandler(update tgbotapi.Update) {
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
	case "chatConfig":
		h.chatConfig(update)
	case "chatSetAgro":
		h.chatSetAgro(update)
	case "chatSetAgroCooldown":
		h.chatSetAgroCooldown(update)
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
	if h.checkAllowed(update.Message.Chat.ID) {
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
}

func (h *Handler) personalHandler(update tgbotapi.Update) {
	if h.checkAllowed(update.Message.Chat.ID) {
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
}

func (h *Handler) listChats(update tgbotapi.Update) {
	defer sentry.Recover()
	if h.isAdmin(update.Message.From.ID) {
		var message string
		for _, chat := range h.chats {
			var lastRand string
			if val, ok := h.chatCache[chat.ID]; !ok {
				lastRand = "never"
			} else {
				lastRand = val.lastRand.Format("2006-01-02 15:04:05")
			}
			message += "/chat " + strconv.FormatInt(chat.ID, 10) +
				"\nChat: " + chat.ChatName +
				"\nType: " + chat.Type +
				"\nLast rand: " + lastRand +
				"\nBilledTo: " + chat.BilledTo.Format("2006-01-02 15:04:05") +
				"\n\n"
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, message)
		msg.ReplyToMessageID = update.Message.MessageID

		_, err := h.bot.Send(msg)
		if err != nil {
			sentry.CaptureException(err)
			log.Println(err)
		}
	}
}

func (h *Handler) chat(update tgbotapi.Update) {
	if h.isAdmin(update.Message.From.ID) {
		id, err := strconv.ParseInt(update.Message.CommandArguments(), 10, 64)
		if err != nil {
			sentry.CaptureException(err)
			log.Println(err)
			return
		}
		chat, err2 := h.db.GetChannelConfig(id)
		if err2 != nil {
			sentry.CaptureException(err2)
			log.Println(err2)
			return
		}

		chatData, err3 := json.Marshal(chat)
		if err3 != nil {
			sentry.CaptureException(err3)
			log.Println(err3)
			return
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, string(chatData))
		msg.ReplyToMessageID = update.Message.MessageID

		_, err4 := h.bot.Send(msg)
		if err4 != nil {
			sentry.CaptureException(err4)
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
				sentry.CaptureException(err)
				log.Println(err)
				return
			}
			chat, err2 := h.db.GetChannelConfig(id)
			if err2 != nil {
				sentry.CaptureException(err2)
				log.Println(err2)
				return
			}

			days, err3 := strconv.Atoi(args[1])
			if err3 != nil {
				sentry.CaptureException(err3)
				log.Println(err3)
				return
			}

			if chat.BilledTo.Before(time.Now()) {
				chat.BilledTo = time.Now().AddDate(0, 0, days)
			} else {
				chat.BilledTo = chat.BilledTo.AddDate(0, 0, days)
			}
			err4 := h.db.UpdateChannelConfig(chat)
			if err4 != nil {
				sentry.CaptureException(err4)
				log.Println(err4)
				return
			}

			h.reloadChannels()

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Done")
			msg.ReplyToMessageID = update.Message.MessageID

			_, err5 := h.bot.Send(msg)
			if err5 != nil {
				sentry.CaptureException(err5)
				log.Println(err5)
			}
		} else {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Wrong arguments")
			msg.ReplyToMessageID = update.Message.MessageID

			_, err := h.bot.Send(msg)
			if err != nil {
				sentry.CaptureException(err)
				log.Println(err)
			}
		}
	}
}

func (h *Handler) chatMakeVIP(update tgbotapi.Update) {
	if h.isAdmin(update.Message.From.ID) {
		id, err := strconv.ParseInt(update.Message.CommandArguments(), 10, 64)
		if err != nil {
			sentry.CaptureException(err)
			log.Println(err)
			return
		}
		chat, err2 := h.db.GetChannelConfig(id)
		if err2 != nil {
			sentry.CaptureException(err2)
			log.Println(err2)
			return
		}

		chat.BilledTo = time.Date(2077, 1, 1, 0, 0, 0, 0, time.UTC)
		err3 := h.db.UpdateChannelConfig(chat)
		if err3 != nil {
			sentry.CaptureException(err3)
			log.Println(err3)
			return
		}

		h.reloadChannels()

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Done")
		msg.ReplyToMessageID = update.Message.MessageID

		_, err4 := h.bot.Send(msg)
		if err4 != nil {
			sentry.CaptureException(err4)
			log.Println(err4)
		}
	}
}

func (h *Handler) chatConfig(update tgbotapi.Update) {
	if h.isChatAdmin(update) {
		idx := slices.IndexFunc(h.chats, func(channel domain.Chat) bool {
			return channel.ID == update.Message.Chat.ID
		})
		if idx == -1 {
			return
		}

		chat := h.chats[idx]

		message := "Chat: " +
			chat.ChatName +
			"\n\nAgro level: " + strconv.Itoa(chat.AgroLevel) + "%" +
			"\n/chatSetAgro <level> - set agro level (chance in %)" +
			"\n0 - disable agro" +
			"\n\nAgro cooldown: " + strconv.Itoa(chat.AgroCooldown) + "min" +
			"\n/chatSetAgroCooldown <cooldown> - set agro cooldown (in minutes). Minimum is 10, max is 1440" +
			"\nIf bot is restarted on server - cooldown will be reset, sorry" +
			"\n\nBilled to: " + chat.BilledTo.Format("2006-01-02 15:04:05")

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, message)
		msg.ReplyToMessageID = update.Message.MessageID

		_, err := h.bot.Send(msg)
		if err != nil {
			sentry.CaptureException(err)
			log.Println(err)
		}
	}
}

func (h *Handler) chatSetAgro(update tgbotapi.Update) {
	if h.isChatAdmin(update) {
		newAgro, err := strconv.Atoi(update.Message.CommandArguments())

		if newAgro < 0 || newAgro > 100 {
			err = errors.New("invalid agro format, use number from 0 to 100")
		}

		if err != nil {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "invalid agro format, use number from 0 to 100")
			msg.ReplyToMessageID = update.Message.MessageID

			_, err2 := h.bot.Send(msg)
			if err2 != nil {
				sentry.CaptureException(err2)
				log.Println(err2)
			}
			return
		}

		chat, err3 := h.db.GetChannelConfig(update.Message.Chat.ID)
		if err3 != nil {
			sentry.CaptureException(err3)
			log.Println(err3)
			return
		}

		chat.AgroLevel = newAgro
		err4 := h.db.UpdateChannelConfig(chat)
		if err4 != nil {
			sentry.CaptureException(err4)
			log.Println(err4)
			return
		}

		h.reloadChannels()

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Done")
		msg.ReplyToMessageID = update.Message.MessageID

		_, err5 := h.bot.Send(msg)
		if err5 != nil {
			sentry.CaptureException(err5)
			log.Println(err5)
		}
	}
}

func (h *Handler) chatSetAgroCooldown(update tgbotapi.Update) {
	if h.isChatAdmin(update) {
		newCooldown, err := strconv.Atoi(update.Message.CommandArguments())

		if newCooldown < 10 || newCooldown > 1440 {
			err = errors.New("invalid agro format, use number from 10 to 1440")
		}

		if err != nil {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "invalid cooldown format, use number from 10 to 1440")
			msg.ReplyToMessageID = update.Message.MessageID

			_, err2 := h.bot.Send(msg)
			if err2 != nil {
				sentry.CaptureException(err2)
				log.Println(err2)
			}
			return
		}

		chat, err3 := h.db.GetChannelConfig(update.Message.Chat.ID)
		if err3 != nil {
			sentry.CaptureException(err3)
			log.Println(err3)
			return
		}

		chat.AgroCooldown = newCooldown
		err4 := h.db.UpdateChannelConfig(chat)
		if err4 != nil {
			sentry.CaptureException(err4)
			log.Println(err4)
			return
		}

		h.reloadChannels()

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Done")
		msg.ReplyToMessageID = update.Message.MessageID

		_, err5 := h.bot.Send(msg)
		if err5 != nil {
			sentry.CaptureException(err5)
			log.Println(err5)
		}
	}
}
