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
	"mvdan.cc/xurls/v2"
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
	ctx := context.Background()
	if update.Message != nil { // If we got a message
		sentry.ConfigureScope(func(scope *sentry.Scope) { scope.SetUser(sentry.User{ID: strconv.Itoa(int(update.Message.From.ID))}) })
		sentry.AddBreadcrumb(&sentry.Breadcrumb{Category: "chat data", Data: map[string]interface{}{"chat id": update.Message.Chat.ID}})
		if h.checkChatExists(update.Message.Chat) {
			switch {
			case update.Message.IsCommand():
				span := sentry.StartSpan(ctx, "command", sentry.WithTransactionName("Handle tg command"))
				h.commandHandler(update)
				span.Finish()
			case h.isPersonal(update):
				span := sentry.StartSpan(ctx, "personal", sentry.WithTransactionName("Handle tg personal message"))
				h.personalHandler(update)
				span.Finish()
			case h.isSupportedURL(update):
				span := sentry.StartSpan(ctx, "personal", sentry.WithTransactionName("Handle not previewed URL"))
				h.fixURLPreview(update)
				span.Finish()
			case h.isItTime(update.Message.Chat.ID):
				span := sentry.StartSpan(ctx, "random", sentry.WithTransactionName("Handle tg random interference"))
				h.randomInterference(update)
				span.Finish()
			}
			ctx.Done()
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
	case "chatSetPreviewDeletion":
		h.chatSetPreviewDeletion(update)
	case "chatUpdateQuestionPrompt":
		h.chatUpdatePrompt(update, "question")
	case "chatUpdateRandomPrompt":
		h.chatUpdatePrompt(update, "random")
	}
}

func (h *Handler) startMessage(update tgbotapi.Update) {
	h.sendMessage(update, helloMessage)
}

func (h *Handler) randomInterference(update tgbotapi.Update) {
	if len(update.Message.Text) > 20 && len(strings.Split(update.Message.Text, " ")) > 3 {
		if h.checkAllowed(update.Message.Chat.ID) {
			h.sendAction(update, tgbotapi.ChatTyping)
			var message string
			ans, err := h.ai.GetPromptResponse(h.promptCompiler(update.Message.Chat.ID, RandomInterference, update))
			if err != nil {
				sentry.CaptureException(err)
				log.Println(err)
				message = openAIErrorMessage
			} else {
				message = ans
			}

			h.sendMessage(update, message)
		}
	}
}

func (h *Handler) personalHandler(update tgbotapi.Update) {
	if h.checkAllowed(update.Message.Chat.ID) {
		if isDraw(update) && len(update.Message.Text) >= 16 && len(strings.Split(update.Message.Text, " ")) >= 2 {
			h.sendAction(update, tgbotapi.ChatUploadPhoto)
			url, err := h.ai.GetImageFromPrompt(getCleanDrawPrompt(update.Message.Text))
			if err != nil {
				sentry.CaptureException(err)
				log.Println(err)
				h.sendMessage(update, openAIErrorMessage)
				return
			}
			h.sendImageByURL(update, url)
		} else {
			h.sendAction(update, tgbotapi.ChatTyping)
			var message string
			ans, err := h.ai.GetPromptResponse(h.promptCompiler(update.Message.Chat.ID, Question, update))
			if err != nil {
				sentry.CaptureException(err)
				log.Println(err)
				message = openAIErrorMessage
			} else {
				message = ans
			}

			h.sendMessage(update, message)
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

		h.sendMessage(update, message)
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

		h.sendMessage(update, string(chatData))
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

			h.sendMessage(update, "Done")
		} else {
			h.sendMessage(update, "Wrong number of arguments")
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

		h.sendMessage(update, "Done")
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
			"\nid: " +
			strconv.FormatInt(chat.ID, 10) +
			"\n\nQuestion prompt: " +
			chat.QuestionPrompt +
			"\n/chatUpdateQuestionPrompt <prompt> (no more than 1000 symbols) - update question prompt" +
			"\n\nRandom interference prompt: " +
			chat.RandomInterferencePrompt +
			"\n/chatUpdateRandomPrompt <prompt> (no more than 1000 symbols) - update random interference prompt" +
			"\n\nAgro level: " + strconv.Itoa(chat.AgroLevel) + "%" +
			"\n/chatSetAgro <level> - set agro level (chance in %)" +
			"\n0 - disable agro" +
			"\n\nAgro cooldown: " + strconv.Itoa(chat.AgroCooldown) + "min" +
			"\n/chatSetAgroCooldown <cooldown> - set agro cooldown (in minutes). Minimum is 10, max is 1440" +
			"\nIf bot is restarted on server - cooldown will be reset, sorry" +
			"\n\n Links preview deletion: " + strconv.FormatBool(chat.DeletePreviewMessages) +
			"\n/chatSetPreviewDeletion <true/false> - set links preview deletion" +
			"\n\nBilled to: " + chat.BilledTo.Format("2006-01-02 15:04:05")

		h.sendMessage(update, message)
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

		h.sendMessage(update, "Done")
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

		h.sendMessage(update, "Done")
	}
}

func (h *Handler) chatSetPreviewDeletion(update tgbotapi.Update) {
	if h.isChatAdmin(update) {
		newDel, err := strconv.ParseBool(update.Message.CommandArguments())

		if err != nil {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "invalid agro format, use `true` or `false`")
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

		chat.DeletePreviewMessages = newDel
		err4 := h.db.UpdateChannelConfig(chat)
		if err4 != nil {
			sentry.CaptureException(err4)
			log.Println(err4)
			return
		}

		h.reloadChannels()

		h.sendMessage(update, "Done")
	}
}

func (h *Handler) chatUpdatePrompt(update tgbotapi.Update, typeOfPrompt string) {
	if h.isChatAdmin(update) {
		chat, err := h.db.GetChannelConfig(update.Message.Chat.ID)
		if err != nil {
			sentry.CaptureException(err)
			log.Println(err)
			return
		}

		if len(update.Message.CommandArguments()) > 1000 {
			h.sendMessage(update, "Prompt is too long, max length is 1000 symbols")
			return
		} else if len(update.Message.CommandArguments()) < 10 {
			h.sendMessage(update, "Prompt is too short, min length is 10 symbols")
			return
		}

		if typeOfPrompt == "question" {
			chat.QuestionPrompt = update.Message.CommandArguments()
		} else {
			chat.RandomInterferencePrompt = update.Message.CommandArguments()
		}
		err2 := h.db.UpdateChannelConfig(chat)
		if err2 != nil {
			sentry.CaptureException(err2)
			log.Println(err2)
			return
		}

		h.reloadChannels()

		h.sendMessage(update, "Done")
	}
}

func (h *Handler) fixURLPreview(update tgbotapi.Update) {
	rxRelaxed := xurls.Relaxed()
	urls := rxRelaxed.FindAllString(update.Message.Text, -1)
	for _, url := range urls {
		if strings.Contains(url, "https://twitter.com") || strings.Contains(url, "https://www.twitter.com") || strings.Contains(url, "https://mobile.twitter.com") {
			h.sendAction(update, tgbotapi.ChatTyping)
			url = strings.ReplaceAll(url, "https://twitter.com", "https://vxtwitter.com")
			url = strings.ReplaceAll(url, "https://www.twitter.com", "https://vxtwitter.com")
			url = strings.ReplaceAll(url, "https://mobile.twitter.com", "https://vxtwitter.com")

			message := "Saved @" + update.Message.From.UserName + " a click:\n" + url
			h.sendMessage(update, message)
			if h.isDeletePreview(update.Message.Chat) {
				h.deleteMessage(update)
			}
		}
		if strings.Contains(url, "https://x.com") || strings.Contains(url, "https://www.x.com") {
			h.sendAction(update, tgbotapi.ChatTyping)
			url = strings.ReplaceAll(url, "https://x.com", "https://vxtwitter.com")
			url = strings.ReplaceAll(url, "https://www.x.com", "https://vxtwitter.com")

			message := "Saved @" + update.Message.From.UserName + " a click:\n" + url
			h.sendMessage(update, message)
			if h.isDeletePreview(update.Message.Chat) {
				h.deleteMessage(update)
			}
		}
		if strings.Contains(url, "https://www.instagram.com") || strings.Contains(url, "https://instagram.com") {
			h.sendAction(update, tgbotapi.ChatTyping)
			url = strings.ReplaceAll(url, "https://www.instagram.com", "https://ddinstagram.com")
			url = strings.ReplaceAll(url, "https://instagram.com", "https://ddinstagram.com")

			message := "Saved @" + update.Message.From.UserName + " a click:\n" + url
			h.sendMessage(update, message)
			if h.isDeletePreview(update.Message.Chat) {
				h.deleteMessage(update)
			}
		}
	}
}
