package tghandler

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/shabablinchikow/nafanya-bot/internal/domain"
)

type ignoredUserStore interface {
	IsUserIgnored(int64, int64) (bool, error)
	SetUserIgnored(int64, int64, bool) error
	ListIgnoredUsers(int64) ([]int64, error)
}

// ignoreFilter runs before all normal message dispatch. Its small dependency
// surface allows testing permissions and dispatch without Telegram or AI calls.
type ignoreFilter struct {
	store       ignoredUserStore
	botUsername string
	isAdmin     func() bool
	reply       func(string)
}

func (h *Handler) filterIgnoredMessage(update tgbotapi.Update) bool {
	f := ignoreFilter{
		store:       h.db,
		botUsername: h.bot.Self.UserName,
		isAdmin:     func() bool { return h.isChatAdmin(update) },
		reply:       func(text string) { h.sendMessage(update, text) },
	}
	return f.handle(update.Message)
}

func ignoreCommand(m *tgbotapi.Message, botUsername string) string {
	if !m.IsCommand() {
		return ""
	}
	full := m.CommandWithAt()
	parts := strings.SplitN(full, "@", 2)
	if len(parts) == 2 && !strings.EqualFold(parts[1], botUsername) {
		return ""
	}
	switch parts[0] {
	case "ignoreme", "unignoreme", "ignoreuser", "unignoreuser", "ignoredusers":
		return parts[0]
	}
	return ""
}

func (f ignoreFilter) handle(m *tgbotapi.Message) bool {
	if m == nil || m.Chat == nil {
		return true
	}
	if m.Chat.Type != domain.ChatTypeGroup && m.Chat.Type != domain.ChatTypeSupergroup {
		return false
	}
	cmd := ignoreCommand(m, f.botUsername)
	// Anonymous admins/channel senders are not an identifiable user. Never
	// persist Telegram's shared anonymous placeholder as a personal opt-out.
	if m.From == nil || m.SenderChat != nil {
		if cmd != "" {
			f.reply("Send this command as your personal account, not anonymously or as a channel.")
			return true
		}
		return m.From == nil
	}
	if cmd != "" {
		f.command(m, cmd)
		return true
	}
	ignored, err := f.store.IsUserIgnored(m.Chat.ID, m.From.ID)
	if err != nil {
		// Fail closed: do not accidentally interact with an opted-out user
		// when the database is unavailable. Do not reply to the sender either.
		log.Printf("checking ignored user: %v", err)
		return true
	}
	return ignored
}

func (f ignoreFilter) command(m *tgbotapi.Message, cmd string) {
	self := cmd == "ignoreme" || cmd == "unignoreme"
	if !self && !f.isAdmin() {
		f.reply("Only group admins can manage other users or view the list. Use /ignoreme or /unignoreme for yourself.")
		return
	}
	if cmd == "ignoredusers" {
		ids, err := f.store.ListIgnoredUsers(m.Chat.ID)
		if err != nil {
			f.dbError(err)
			return
		}
		if len(ids) == 0 {
			f.reply("No ignored users in this group.")
			return
		}
		// Chunk large lists below Telegram's message length limit.
		text := "Ignored users in this group:"
		for _, id := range ids {
			line := "\n" + strconv.FormatInt(id, 10)
			if len(text)+len(line) > 3500 {
				f.reply(text)
				text = "Ignored users (continued):"
			}
			text += line
		}
		f.reply(text)
		return
	}
	id := m.From.ID
	arg := strings.TrimSpace(m.CommandArguments())
	if self {
		if arg != "" {
			f.reply("Use /" + cmd + " without arguments; it only changes your own setting.")
			return
		}
	} else {
		var err error
		if arg != "" {
			id, err = strconv.ParseInt(arg, 10, 64)
		} else if r := m.ReplyToMessage; r != nil && r.From != nil && r.SenderChat == nil {
			id = r.From.ID
		} else {
			id = 0
		}
		if err != nil || id <= 0 {
			f.reply("Use /" + cmd + " <positive numeric user ID>, or reply to a user's message without arguments.")
			return
		}
	}
	ignored := cmd == "ignoreme" || cmd == "ignoreuser"
	if err := f.store.SetUserIgnored(m.Chat.ID, id, ignored); err != nil {
		f.dbError(err)
		return
	}
	if ignored {
		f.reply(fmt.Sprintf("User %d is now ignored in this group. Use /unignoreme to opt back in.", id))
	} else {
		f.reply(fmt.Sprintf("User %d is no longer ignored in this group.", id))
	}
}

func (f ignoreFilter) dbError(err error) {
	log.Printf("updating ignored users: %v", err)
	f.reply("Could not access ignore settings. Please try again later.")
}
