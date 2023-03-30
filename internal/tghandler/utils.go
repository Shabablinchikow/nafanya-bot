package tghandler

import (
	"crypto/rand"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"math/big"
	"strings"
	"time"
)

type ChatCache struct {
	lastRand time.Time
}

var cache = make(map[int64]ChatCache)

func isItTime(chat int64) bool {
	nBig, err := rand.Int(rand.Reader, big.NewInt(100))
	if err != nil {
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

func isPersonal(update tgbotapi.Update) bool {
	return strings.HasPrefix(update.Message.Text, "Нафаня") || strings.HasPrefix(update.Message.Text, "нафаня")
}
