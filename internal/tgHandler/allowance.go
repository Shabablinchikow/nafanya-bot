package tgHandler

var allowedChats = []int64{
	438663,
	189049,
	89895968,
	-1720840717,
	-934228897,
	-1001720840717,
	886350649,
}

func checkAllowed(ID int64) bool {
	for _, chatID := range allowedChats {
		if chatID == ID {
			return true
		}
	}
	return false
}
