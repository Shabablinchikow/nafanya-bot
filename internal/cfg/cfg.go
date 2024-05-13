package cfg

import (
	"encoding/base64"
	"github.com/getsentry/sentry-go"
	"os"
	"strconv"
)

type Cfg struct {
	BotToken     string
	OAIToken     string
	GoogleToken  string
	DefaultAdmin int64

	DBHost   string
	DBPort   string
	DBUser   string
	DBPass   string
	DBName   string
	DBSSL    string
	DBPrefix string

	SentryDSN string
	DebugMode bool
}

// LoadConfig loads the config from the environment variables
func LoadConfig() Cfg {
	var cfg Cfg

	token, err := base64.StdEncoding.DecodeString(fillEnv("GEMINI_API_KEY"))
	if err != nil {
		sentry.CaptureException(err)
		panic(err)

	}

	cfg.BotToken = fillEnv("BOT_TOKEN")
	cfg.OAIToken = fillEnv("AI_TOKEN")
	cfg.GoogleToken = string(token)

	adminID, err := strconv.ParseInt(getEnv("DEFAULT_ADMIN", "438663"), 10, 64)
	if err != nil {
		sentry.CaptureException(err)
		panic(err)
	}
	cfg.DefaultAdmin = adminID

	cfg.DBHost = fillEnv("DB_HOST")
	cfg.DBPort = fillEnv("DB_PORT")
	cfg.DBUser = fillEnv("DB_USER")
	cfg.DBPass = fillEnv("DB_PASS")
	cfg.DBName = fillEnv("DB_NAME")
	cfg.DBSSL = getEnv("DB_SSL", "disable")
	cfg.DBPrefix = getEnv("DB_PREFIX", "nafanya_")

	cfg.SentryDSN = getEnv("SENTRY_DSN", "")
	cfg.DebugMode = getEnv("DEBUG_MODE", "false") == "true"

	return cfg
}

func fillEnv(key string) string {
	value, ok := os.LookupEnv(key)
	if ok {
		return value
	}
	panic(key + " not set")
}

// getEnv returns the value of the environment variable or the fallback value
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
