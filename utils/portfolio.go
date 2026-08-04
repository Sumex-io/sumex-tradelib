package utils

import (
	"strings"
	"time"
)

// StartOfUTCDay возвращает начало суток UTC для времени t
func StartOfUTCDay(t time.Time) time.Time {
	return time.Date(t.UTC().Year(), t.UTC().Month(), t.UTC().Day(), 0, 0, 0, 0, time.UTC)
}

// StartOfUTCWeekMonday возвращает понедельник 00:00 UTC для времени t
func StartOfUTCWeekMonday(t time.Time) time.Time {
	t = StartOfUTCDay(t)
	wd := int(t.Weekday()) // 0=Sunday ... 1=Monday
	if wd == 0 {
		wd = 7
	}
	return t.AddDate(0, 0, -(wd - 1))
}

// SizeFromNotionalAndEntry рассчитывает размер (в базовой монете) из нотионала и цены входа
func SizeFromNotionalAndEntry(notional, entry float64) float64 {
	if entry <= 0 {
		return 0
	}
	return notional / entry
}

// DeriveTPFromBotSettings считает TP цену по entry и takeProfit (в %)
func DeriveTPFromBotSettings(entry float64, posSide string, takeProfit float64) float64 {
	if entry <= 0 || takeProfit <= 0 {
		return 0
	}
	switch strings.ToUpper(strings.TrimSpace(posSide)) {
	case "LONG":
		return entry * (1 + takeProfit/100.0)
	case "SHORT":
		return entry * (1 - takeProfit/100.0)
	default:
		return 0
	}
}
