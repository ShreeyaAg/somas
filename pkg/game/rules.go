package game

import (
	"math"
)

func Clamp(n, lo, hi float64) float64 {
	if math.IsNaN(n) || math.IsInf(n, 0) {
		return lo
	}
	return max(lo, min(n, hi))
}
func Productivity(w int, c Config) float64 {
	n := float64(min(w, c.YieldCap))
	return n*(c.YieldBase+c.YieldSlope*n) + float64(max(0, w-c.YieldCap))*c.YieldExtra
}
func Multiplier(ctk float64, c Config) float64 {
	if ctk < c.Thresholds[0] {
		return 0
	}
	if ctk < c.Thresholds[1] {
		return .5
	}
	if ctk < c.Thresholds[2] {
		return 1
	}
	return 1.5
}
