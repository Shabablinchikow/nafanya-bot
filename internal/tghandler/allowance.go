package tghandler

var allowedChats = []int64{
	438663,
	189049,
	89895968,
	-1720840717,
	-934228897,
	-1001720840717,
	886350649,
	1965353629,
}

func checkAllowed(id int64) bool {
	for _, chatID := range allowedChats {
		if chatID == id {
			return true
		}
	}
	return false
}
