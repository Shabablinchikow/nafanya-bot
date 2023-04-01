package tghandler

import (
	"crypto/rand"
	"github.com/getsentry/sentry-go"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/shabablinchikow/nafanya-bot/internal/domain"
	"golang.org/x/exp/slices"
	"math/big"
	"strings"
	"time"
)

const (
	Question           = 1
	RandomInterference = 2
)

type chatCache struct {
	lastRand time.Time
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

func (h *Handler) isItTime(chat int64) bool {
	defer sentry.Recover()

	idx := slices.IndexFunc(h.chats, func(channel domain.Chat) bool {
		return channel.ID == chat
	})
	if idx == -1 {
		return false
	}

	if h.chats[idx].Type == domain.ChatTypePrivate {
		return false
	}

	nBig, err := rand.Int(rand.Reader, big.NewInt(100))
	if err != nil {
		sentry.CaptureException(err)
		panic(err)
	}
	n := nBig.Int64()

	if _, ok := h.chatCache[chat]; !ok {
		newCache := chatCache{lastRand: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)}
		h.chatCache[chat] = newCache
	}

	agroLevel := int64(h.chats[idx].AgroLevel)
	cooldown := time.Duration(h.chats[idx].AgroCooldown)

	if n > (100-agroLevel) && time.Since(h.chatCache[chat].lastRand) > (cooldown*time.Minute) {
		newCache := chatCache{lastRand: time.Now()}
		h.chatCache[chat] = newCache
		return true
	}

	return false
}

func (h *Handler) isPersonal(update tgbotapi.Update) bool {
	if strings.HasPrefix(update.Message.Text, "Нафаня") || strings.HasPrefix(update.Message.Text, "нафаня") {
		return true
	} else if update.Message.ReplyToMessage != nil {
		return update.Message.ReplyToMessage.From.ID == h.bot.Self.ID
	}
	return false
}

func rollEmotion() string {
	nBig, err := rand.Int(rand.Reader, big.NewInt(int64(len(emotionList))))
	if err != nil {
		sentry.CaptureException(err)
		panic(err)
	}
	n := nBig.Int64()

	return emotionList[n]
}

func (h *Handler) promptCompiler(id int64, promptType int, update tgbotapi.Update) (prompt string, userInput string) {
	idx := slices.IndexFunc(h.chats, func(channel domain.Chat) bool {
		return channel.ID == id
	})

	curChannel := h.chats[idx]

	userInput = update.Message.From.FirstName + " " + update.Message.From.LastName + ": " + update.Message.Text

	nextMess := update.Message.ReplyToMessage

	if nextMess != nil {
		if !nextMess.From.IsBot {
			userInput = nextMess.From.FirstName + " " + nextMess.From.LastName + ": " + nextMess.Text + "\n" + userInput
		} else {
			userInput = nextMess.From.FirstName + ": " + nextMess.Text + "\n" + userInput
		}
	}

	switch promptType {
	case Question:
		prompt = strings.ReplaceAll(curChannel.QuestionPrompt, "{emotion}", rollEmotion())
	case RandomInterference:
		prompt = strings.ReplaceAll(curChannel.RandomInterferencePrompt, "{emotion}", rollEmotion())
	}

	return prompt, userInput
}

func (h *Handler) reloadChannels() {
	var err error
	h.chats, err = h.db.GetAllChannelsConfig()
	if err != nil {
		sentry.CaptureException(err)
		panic(err)
	}
}