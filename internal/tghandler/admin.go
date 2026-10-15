package tghandler

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/shabablinchikow/nafanya-bot/internal/domain"
)

func (h *Handler) isAdmin(id int64) bool {
	for _, admin := range h.config.Admins {
		if id == admin {
			return true
		}
	}

	return false
}

func (h *Handler) isChatAdmin(update tgbotapi.Update) bool {
	m := update.Message
	if m == nil || m.Chat == nil || m.From == nil || m.SenderChat != nil {
		return false
	}
	// Authorization must work before the first message registers a new chat.
	// Use Telegram's incoming chat type and live administrator list, not h.chats.
	if m.Chat.Type == domain.ChatTypePrivate {
		return true
	}

	admins, err := h.bot.GetChatAdministrators(tgbotapi.ChatAdministratorsConfig{ChatConfig: tgbotapi.ChatConfig{ChatID: update.Message.Chat.ID}})
	if err != nil {
		return false
	}

	for _, admin := range admins {
		if admin.User.ID == update.Message.From.ID {
			return true
		}
	}

	return false
}
