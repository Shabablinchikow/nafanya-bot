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

type ChatCache struct {
	lastRand time.Time
}

var cache = make(map[int64]ChatCache)

// emotionList is a list of strings containing all available emotions
var emotionList = []string{
	"с нейтральным отношением",
	"с пессимизмом",
	"с оптимизмом",
	"с сарказмом",
	"с раздражением",
	"с жестким негативом",
}

func isItTime(chat int64) bool {
	nBig, err := rand.Int(rand.Reader, big.NewInt(100))
	if err != nil {
		sentry.CaptureException(err)
		panic(err)
	}
	n := nBig.Int64()

	if n > 95 && time.Since(cache[chat].lastRand) > 10*time.Minute {
		newCache := ChatCache{time.Now()}
		cache[chat] = newCache
		return true
	}

	return false
}

func (h *Handler) isPersonal(update tgbotapi.Update) bool {
	return strings.HasPrefix(update.Message.Text, "Нафаня") || strings.HasPrefix(update.Message.Text, "нафаня") || update.Message.ReplyToMessage.From.ID == h.bot.Self.ID
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
