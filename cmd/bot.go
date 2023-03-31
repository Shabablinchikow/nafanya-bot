package main

import (
	"github.com/getsentry/sentry-go"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sashabaranov/go-openai"
	"github.com/shabablinchikow/nafanya-bot/internal/aihandler"
	"github.com/shabablinchikow/nafanya-bot/internal/cfg"
	"github.com/shabablinchikow/nafanya-bot/internal/domain"
	"github.com/shabablinchikow/nafanya-bot/internal/tghandler"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"log"
	"time"
)

func main() {
	// Load the config from the environment variables
	config := cfg.LoadConfig()

	var env string
	if config.DebugMode {
		env = "development"
	} else {
		env = "production"
	}

	errSentry := sentry.Init(sentry.ClientOptions{
		Dsn:              config.SentryDSN,
		AttachStacktrace: true,
		TracesSampleRate: 1.0,
		EnableTracing:    true,
		Environment:      env,
	})
	if errSentry != nil {
		log.Fatalf("sentry.Init: %s", errSentry)
	}

	defer sentry.Recover()
	defer sentry.Flush(2 * time.Second)

	ai := openai.NewClient(config.AIToken)
	aiHndlr := aihandler.NewHandler(ai)

	dbDSN := "host=" + config.DBHost + " user=" + config.DBUser + " password=" + config.DBPass + " dbname=" + config.DBName + " port=" + config.DBPort + " sslmode=" + config.DBSSL
	dbConfig := &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix: config.DBPrefix,
		},
	}
	db, err := domain.NewHandler(dbDSN, dbConfig)

	if err != nil {
		sentry.CaptureException(err)
		log.Panic(err)
	}

	bot, err2 := tgbotapi.NewBotAPI(config.BotToken)
	if err2 != nil {
		sentry.CaptureException(err2)
		log.Panic(err2)
	}

	bot.Debug = config.DebugMode

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	handler := tghandler.NewHandler(bot, aiHndlr, db)

	for update := range updates {
		handler.HandleEvents(update)
	}
}
