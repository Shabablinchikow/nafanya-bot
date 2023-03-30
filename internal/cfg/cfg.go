package cfg

import "os"

type Cfg struct {
	BotToken string
	AIToken  string

	DBHost   string
	DBPort   string
	DBUser   string
	DBPass   string
	DBName   string
	DBSSL    string
	DBPrefix string

	DebugMode bool
}

// LoadConfig loads the config from the environment variables
func LoadConfig() Cfg {
	var cfg Cfg

	cfg.BotToken = fillEnv("BOT_TOKEN")
	cfg.AIToken = fillEnv("AI_TOKEN")
	cfg.DBHost = fillEnv("DB_HOST")
	cfg.DBPort = fillEnv("DB_PORT")
	cfg.DBUser = fillEnv("DB_USER")
	cfg.DBPass = fillEnv("DB_PASS")
	cfg.DBName = fillEnv("DB_NAME")
	cfg.DBSSL = getEnv("DB_SSL", "disable")
	cfg.DBPrefix = getEnv("DB_PREFIX", "nafanya_")

	cfg.DebugMode = getEnv("DEBUG_MODE", "false") == "true"

	return cfg
}

func fillEnv(key string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	} else {
		panic(key + " not set")
	}
}

// getEnv returns the value of the environment variable or the fallback value
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
