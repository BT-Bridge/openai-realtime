package tools

import "time"

func FrameSamples(duration time.Duration, rate, channels int) int {
	return int(duration.Seconds() * float64(channels) * float64(rate))
}
