package domain

import (
	"gorm.io/gorm"
	"time"
)

const (
	ChatTypeGroup      = "group"
	ChatTypePrivate    = "private"
	ChatTypeSupergroup = "supergroup"
	ChatTypeChannel    = "channel"
)

type Channel struct {
	gorm.Model
	ID                       int64     `gorm:"primaryKey"`
	Type                     string    `gorm:"type:varchar(20)"`
	ChannelName              string    `gorm:"type:varchar(255)"`
	QuestionPrompt           string    `gorm:"type:varchar(255)"`
	RandomInterferencePrompt string    `gorm:"type:varchar(255)"`
	EmotionsEnable           bool      `gorm:"type:bool"`
	AgroLevel                int       `gorm:"type:int"`
	AgroCooldown             int       `gorm:"type:int"`
	BilledTo                 time.Time `gorm:"type:timestamp"`
}
