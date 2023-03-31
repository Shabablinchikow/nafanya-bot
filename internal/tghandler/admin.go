package tghandler

func (h *Handler) isAdmin(id int64) bool {
	for _, admin := range h.config.Admins {
		if admin == id {
			return true
		}
	}

	return false
}
