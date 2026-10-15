package domain

import "gorm.io/gorm/clause"

// IgnoredUser is scoped to a chat. A composite key makes repeated/concurrent
// opt-outs idempotent, without read/modify/write races on chat configuration.
// No soft deletion: users opting back in are removed completely.
type IgnoredUser struct {
	ChatID int64 `gorm:"primaryKey;autoIncrement:false"`
	UserID int64 `gorm:"primaryKey;autoIncrement:false"`
}

func (h *Handler) IsUserIgnored(chatID, userID int64) (bool, error) {
	var count int64
	err := h.db.Model(&IgnoredUser{}).Where("chat_id = ? AND user_id = ?", chatID, userID).Count(&count).Error
	return count > 0, err
}

func (h *Handler) SetUserIgnored(chatID, userID int64, ignored bool) error {
	row := IgnoredUser{ChatID: chatID, UserID: userID}
	if ignored {
		return h.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error
	}
	return h.db.Where("chat_id = ? AND user_id = ?", chatID, userID).Delete(&IgnoredUser{}).Error
}

func (h *Handler) ListIgnoredUsers(chatID int64) ([]int64, error) {
	var ids []int64
	err := h.db.Model(&IgnoredUser{}).Where("chat_id = ?", chatID).Order("user_id").Pluck("user_id", &ids).Error
	return ids, err
}
