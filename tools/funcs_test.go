package tools

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFrameSamples(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		rate     int
		channels int
		expected int
	}{
		{
			name:     "Basic stereo at 48kHz for 120ms",
			duration: 120 * time.Millisecond,
			rate:     48000,
			channels: 2,
			expected: 11520, // 0.12s * 48000 * 2 = 11520
		},
		{
			name:     "Mono at 44.1kHz for 1s",
			duration: time.Second,
			rate:     44100,
			channels: 1,
			expected: 44100,
		},
		{
			name:     "Stereo at 48kHz for 20ms",
			duration: 20 * time.Millisecond,
			rate:     48000,
			channels: 2,
			expected: 1920, // 0.02s * 48000 * 2 = 1920
		},
		{
			name:     "Zero duration",
			duration: 0,
			rate:     48000,
			channels: 2,
			expected: 0,
		},
		{
			name:     "Zero channels",
			duration: time.Second,
			rate:     48000,
			channels: 0,
			expected: 0,
		},
		{
			name:     "Zero rate",
			duration: time.Second,
			rate:     0,
			channels: 2,
			expected: 0,
		},
		{
			name:     "Large values",
			duration: 10 * time.Second,
			rate:     96000,
			channels: 4,
			expected: 3840000, // 10s * 96000 * 4 = 3,840,000
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FrameSamples(tt.duration, tt.rate, tt.channels)
			assert.Equal(t, tt.expected, result)
		})
	}
}
