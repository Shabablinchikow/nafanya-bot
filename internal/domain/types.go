package domain

import (
	pq "github.com/lib/pq"
	"gorm.io/gorm"
	"time"
)

const (
	ChatTypeGroup      = "group"
	ChatTypePrivate    = "private"
	ChatTypeSupergroup = "supergroup"
	ChatTypeChannel    = "channel"

	DefaultQuestionPrompt     = "Тебя зовут Нафаня. Ты бот-помощник в онлайн чате. Твоя задача — отвечать на вопросы участников чата {emotion}, можно материться, но не более 140 слов. Далее будет фрагмент диалога из чата. Начинай свой ответ с новой строки в роли Нафани. Не приветствуй участника чата в начале сообщения."
	DefaultInterferencePrompt = "Тебя зовут Нафаня. Ты учачтник онлайн чата. Твоя задача – вклиниваться в диалог участников чата {emotion}, но не более 80 слов. Далее будет фрагмент диалога из чата. Начинай свой ответ с новой строки в роли Нафани. Не приветствуй участника чата в начале сообщения."
)

type Chat struct {
	gorm.Model
	ID                       int64     `gorm:"primaryKey"`
	Type                     string    `gorm:"type:varchar(20)"`
	ChatName                 string    `gorm:"type:varchar(255)"`
	QuestionPrompt           string    `gorm:"type:text"`
	RandomInterferencePrompt string    `gorm:"type:text"`
	EmotionsEnable           bool      `gorm:"type:bool"`
	DeletePreviewMessages    bool      `gorm:"type:bool"`
	AgroLevel                int       `gorm:"type:int"` // chance in percent of random interference
	AgroCooldown             int       `gorm:"type:int"` // cooldown in minutes between random interference
	BilledTo                 time.Time `gorm:"type:timestamp"`
	AIModel                  string    `gorm:"type:text"`
}

type BotConfig struct {
	gorm.Model
	Admins          pq.Int64Array `gorm:"type:bigint[]"`
	GoogleMaxTokens int           `gorm:"type:int"`
	OAIMaxTokens    int           `gorm:"type:int"`
}

func GetDefaultChat() Chat {
	return Chat{
		ID:                       1,
		Type:                     ChatTypePrivate,
		ChatName:                 "Chat",
		QuestionPrompt:           DefaultQuestionPrompt,
		RandomInterferencePrompt: DefaultInterferencePrompt,
		EmotionsEnable:           true,
		AgroLevel:                5,
		AgroCooldown:             10,
		BilledTo:                 time.Now(),
	}
}
