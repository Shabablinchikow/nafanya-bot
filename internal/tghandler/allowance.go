package tghandler

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/shabablinchikow/nafanya-bot/internal/domain"
	"golang.org/x/exp/slices"
	"time"
)

func (h *Handler) checkAllowed(id int64) bool {
	idx := slices.IndexFunc(h.chats, func(channel domain.Chat) bool {
		return channel.ID == id
	})

	if idx == -1 || h.chats[idx].BilledTo.Before(time.Now()) {
		return false
	}

	return true
}

func (h *Handler) checkChatExists(chat *tgbotapi.Chat) bool {
	idx := slices.IndexFunc(h.chats, func(channel domain.Chat) bool {
		return channel.ID == chat.ID
	})

	return idx != -1
}

func (h *Handler) isDeletePreview(chat *tgbotapi.Chat) bool {
	idx := slices.IndexFunc(h.chats, func(channel domain.Chat) bool {
		return channel.ID == chat.ID
	})

	if idx == -1 {
		return false
	}

	return h.chats[idx].DeletePreviewMessages
}
