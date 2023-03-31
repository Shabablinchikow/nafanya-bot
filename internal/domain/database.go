package domain

import (
	"github.com/getsentry/sentry-go"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Handler struct {
	db *gorm.DB
}

func NewHandler(dsn string, config gorm.Option) (*Handler, error) {
	db, err := gorm.Open(postgres.Open(dsn), config)
	if err != nil {
		sentry.CaptureException(err)
		return nil, err
	}

	// TODO: fix migration
	/*err2 := db.AutoMigrate(&Channel{})
	if err2 != nil {
		panic(err2)
	}*/

	return &Handler{db}, nil
}

func (h *Handler) GetAllChannelsConfig() ([]Channel, error) {
	var channels []Channel
	if err := h.db.Find(&channels).Error; err != nil {
		return nil, err
	}

	return channels, nil
}

func (h *Handler) GetChannelConfig(id int64) (Channel, error) {
	var channel Channel
	if err := h.db.First(&channel, id).Error; err != nil {
		return Channel{}, err
	}

	return channel, nil
}

func (h *Handler) CreateChannelConfig(channel Channel) error {
	if err := h.db.Create(&channel).Error; err != nil {
		return err
	}

	return nil
}
