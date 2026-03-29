package util

import "fmt"

func FormatUSDMinor(amount int64) string {
	sign := ""
	if amount < 0 {
		sign = "-"
		amount = -amount
	}
	return fmt.Sprintf("%s$%d.%02d", sign, amount/100, amount%100)
}
