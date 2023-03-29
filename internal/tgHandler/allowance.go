package tgHandler

var allowedChats = []int64{
	438663,
	189049,
	89895968,
	-1720840717,
}

func checkAllowed(ID int64) bool {
	for _, chatID := range allowedChats {
		if chatID == ID {
			return true
		}
	}
	return false
}
