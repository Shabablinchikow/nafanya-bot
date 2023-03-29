package main

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	openai "github.com/sashabaranov/go-openai"
	"github.com/shabablinchikow/nafanya-bot/internal/aiHandler"
	"github.com/shabablinchikow/nafanya-bot/internal/cfg"
	"github.com/shabablinchikow/nafanya-bot/internal/tgHandler"

	"log"
)

func main() {
	// Load the config from the environment variables
	config := cfg.LoadConfig()

	ai := openai.NewClient(config.AIToken)
	aiHndlr := aiHandler.NewHandler(ai)

	bot, err := tgbotapi.NewBotAPI(config.BotToken)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = config.DebugMode

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	handler := tgHandler.NewHandler(bot, aiHndlr)

	for update := range updates {
		handler.HandleEvents(update)
	}
}
