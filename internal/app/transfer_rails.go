package app

import "strings"

func NormalizeTransferRail(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "internal", "account_transfer":
		return "account"
	case "rtp", "real-time-payments":
		return "real_time_payments"
	default:
		return strings.TrimSpace(strings.ToLower(value))
	}
}
