package models

import (
	"fmt"
	"math"
)

// BaselineValue computes the long-run market valuation anchor in EUR.
// Fitted to the dataset curve (~x1.086 per OVR) with youth/wonderkid premiums
// and veteran depreciation past 32.
func BaselineValue(ovr int, age int, isWonderkid bool) int64 {
	ovrDiff := ovr - 65
	if ovrDiff < 0 {
		ovrDiff = 0
	}
	base := 11000000.0 * math.Pow(1.086, float64(ovrDiff))

	if age <= 21 {
		base *= 1.15
	}
	if isWonderkid {
		base *= 1.35
	}
	if age >= 33 {
		base *= math.Pow(0.75, float64(age-32))
	}

	if base < 500000.0 {
		return 500000
	}
	return int64(math.Round(base))
}

// ClampValue bounds market valuation within a dynamic corridor around its anchor
// [0.35 * anchor, 3.0 * anchor] and absolute hard bounds [€300k, €500M].
func ClampValue(current int64, ovr int, age int, isWonderkid bool) int64 {
	if current < 0 {
		current = 0
	}
	anchor := float64(BaselineValue(ovr, age, isWonderkid))
	lowCorridor := math.Round(anchor * 0.35)
	highCorridor := math.Round(anchor * 3.0)

	val := float64(current)
	if val > highCorridor {
		val = highCorridor
	}
	if val < lowCorridor {
		val = lowCorridor
	}
	if val > 500000000.0 {
		val = 500000000.0
	}
	if val < 300000.0 {
		val = 300000.0
	}
	return int64(math.Round(val))
}

// ClampPlayer calculates and updates the player's market value inside safe guardrails.
func ClampPlayer(p *Player) int64 {
	if p == nil {
		return 0
	}
	p.MarketValueEUR = ClampValue(p.MarketValueEUR, p.OVR, p.Age, p.UniverseWonderkid)
	return p.MarketValueEUR
}

// WageForOVR returns the baseline weekly wage anchor in EUR derived from overall rating.
func WageForOVR(ovr int) int64 {
	ovrDiff := ovr - 60
	if ovrDiff < 0 {
		ovrDiff = 0
	}
	return int64(math.Round(2000.0 * math.Pow(1.23, float64(ovrDiff))))
}

// FormatCurrency formats a euro amount into a readable string (€T, €B, €M, €K, or exact €).
func FormatCurrency(valueEUR int64) string {
	if valueEUR >= 1000000000000 {
		return fmt.Sprintf("€%.2fT", float64(valueEUR)/1e12)
	}
	if valueEUR >= 1000000000 {
		return fmt.Sprintf("€%.2fB", float64(valueEUR)/1e9)
	}
	if valueEUR >= 1000000 {
		return fmt.Sprintf("€%.1fM", float64(valueEUR)/1e6)
	}
	if valueEUR >= 1000 {
		return fmt.Sprintf("€%dK", valueEUR/1000)
	}
	return fmt.Sprintf("€%d", valueEUR)
}

// FormatWage formats weekly wage into a clean string (€M/wk or €k/wk).
func FormatWage(wageEUR int64) string {
	if wageEUR >= 1000000 {
		return fmt.Sprintf("€%.2fM/wk", float64(wageEUR)/1e6)
	}
	return fmt.Sprintf("€%dk/wk", wageEUR/1000)
}
