package domain

import (
	"github.com/getsentry/sentry-go"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
)

// legacyModels maps model IDs from older releases to their current cfg.AIModel*/cfg.ImageModel* value.
var legacyModels = map[string]string{
	"gpt-5.5":                "gpt-6.2-sol",
	"gemini-3.5-flash":       "gemini-3.8-flash",
	"deepseek-v4-pro":        "deepseek-flash",
	"gpt-image-2":            "gpt-image-2.6-sunburst",
	"gemini-3.1-flash-image": "gemini-nano-banana-2.1",
}

type Handler struct {
	db *gorm.DB
}

func NewHandler(dsn string, config gorm.Option, defaultAdmin int64) (*Handler, error) {
	db, err := gorm.Open(postgres.Open(dsn), config)
	if err != nil {
		sentry.CaptureException(err)
		return nil, err
	}

	// ponytail: shared RDB has max_connections=100; gorm's pool is otherwise unbounded.
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(5)

	err2 := db.AutoMigrate(&Chat{}, &BotConfig{}, &IgnoredUser{})
	if err2 != nil {
		panic(err2)
	}

	// Rewrite model IDs stored by older versions so chats keep their chosen provider.
	for old, cur := range legacyModels {
		db.Model(&Chat{}).Where("ai_model = ?", old).Update("ai_model", cur)
		db.Model(&Chat{}).Where("image_model = ?", old).Update("image_model", cur)
	}

	log.Println("Migrated")

	// create default bot config if it doesn't exist
	var rowCount int64
	db.Model(&BotConfig{}).Count(&rowCount)
	if rowCount == 0 {
		log.Println("added admin")
		db.Create(&BotConfig{
			Admins: []int64{defaultAdmin},
		})
	}

	return &Handler{db}, nil
}

func (h *Handler) GetAllChannelsConfig() ([]Chat, error) {
	var channels []Chat
	if err := h.db.Find(&channels).Error; err != nil {
		return nil, err
	}

	return channels, nil
}

func (h *Handler) GetChannelConfig(id int64) (Chat, error) {
	var channel Chat
	if err := h.db.First(&channel, id).Error; err != nil {
		return Chat{}, err
	}

	return channel, nil
}

func (h *Handler) CreateChannelConfig(channel Chat) error {
	return h.db.Create(&channel).Error
}

func (h *Handler) UpdateChannelConfig(channel Chat) error {
	return h.db.Save(&channel).Error
}

func (h *Handler) GetBotConfig() (BotConfig, error) {
	var botConfig BotConfig
	err := h.db.First(&botConfig).Error

	return botConfig, err
}

func (h *Handler) AddAdmin(id int64) error {
	currentConfig, err := h.GetBotConfig()
	if err != nil {
		sentry.CaptureException(err)
		return err
	}

	currentConfig.Admins = append(currentConfig.Admins, id)
	return h.db.Updates(&currentConfig).Error
}

func (h *Handler) GetMaxTokens() (GoogleMaxTokens int, OAIMaxTokens int) {
	var botConfig BotConfig
	err := h.db.First(&botConfig).Error
	if err != nil {
		sentry.CaptureException(err)
		return 0, 0
	}

	return botConfig.GoogleMaxTokens, botConfig.OAIMaxTokens
}

func (h *Handler) UpdateMaxTokens(maxTokens int) error {
	var botConfig BotConfig
	err := h.db.First(&botConfig).Error
	if err != nil {
		sentry.CaptureException(err)
		return err
	}

	botConfig.GoogleMaxTokens = maxTokens
	botConfig.OAIMaxTokens = maxTokens

	return h.db.Save(&botConfig).Error
}
