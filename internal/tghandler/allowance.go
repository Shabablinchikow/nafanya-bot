package tghandler

import (
	"github.com/shabablinchikow/nafanya-bot/internal/domain"
	"golang.org/x/exp/slices"
	"time"
)

func (h *Handler) checkAllowed(id int64) bool {
	idx := slices.IndexFunc(h.channels, func(channel domain.Channel) bool {
		return channel.ID == id
	})

	if idx == -1 || h.channels[idx].BilledTo.Before(time.Now()) {
		return false
	}

	return true
}
