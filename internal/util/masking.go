package util

func MaskLast4(value string) string {
	if len(value) <= 4 {
		return value
	}
	return "****" + value[len(value)-4:]
}

func MaskAccountNumber(value string) string {
	return MaskLast4(value)
}
