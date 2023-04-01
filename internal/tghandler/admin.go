package tghandler

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/shabablinchikow/nafanya-bot/internal/domain"
	"golang.org/x/exp/slices"
)

func (h *Handler) isAdmin(id int64) bool {
	for _, admin := range h.config.Admins {
		if admin == id {
			return true
		}
	}

	return false
}

func (h *Handler) isChatAdmin(update tgbotapi.Update) bool {
	idx := slices.IndexFunc(h.chats, func(channel domain.Chat) bool {
		return channel.ID == update.Message.Chat.ID
	})
	if idx == -1 {
		return false
	}

	if h.chats[idx].Type == domain.ChatTypePrivate {
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
