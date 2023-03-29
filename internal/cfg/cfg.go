package cfg

import "os"

type Cfg struct {
	BotToken  string
	AIToken   string
	DebugMode bool
}

// LoadConfig loads the config from the environment variables
func LoadConfig() Cfg {
	var cfg Cfg

	if value, ok := os.LookupEnv("BOT_TOKEN"); ok {
		cfg.BotToken = value
	} else {
		panic("BOT_TOKEN not set")
	}

	if value, ok := os.LookupEnv("AI_TOKEN"); ok {
		cfg.AIToken = value
	} else {
		panic("AI_TOKEN not set")
	}

	cfg.DebugMode = getEnv("DEBUG_MODE", "false") == "true"

	return cfg
}

// getEnv returns the value of the environment variable or the fallback value
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
