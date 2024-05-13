package domain

import (
	"github.com/getsentry/sentry-go"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
)

type Handler struct {
	db *gorm.DB
}

func NewHandler(dsn string, config gorm.Option, defaultAdmin int64) (*Handler, error) {
	db, err := gorm.Open(postgres.Open(dsn), config)
	if err != nil {
		sentry.CaptureException(err)
		return nil, err
	}

	err2 := db.AutoMigrate(&Chat{}, &BotConfig{})
	if err2 != nil {
		panic(err2)
	}

	log.Println("Migrated")

	// create default bot config if it doesn't exist
	var rowCount int64
	db.Find(&BotConfig{}).Count(&rowCount)
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
