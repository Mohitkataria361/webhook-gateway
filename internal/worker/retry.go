package worker

import (
	"math"
	"math/rand"
	"time"
)

// Backoff computes the exponential backoff delay with jitter for a given attempt.
//
//	Delay = base_delay * 2^attempt + jitter
//
// Jitter is a random value in [0, base_delay] to avoid thundering-herd
// when many workers retry simultaneously.
func Backoff(attempt int) time.Duration {
	const baseDelayMS = 1000
	const maxDelayMS = 30000 // Cap at 30s

	exp := math.Pow(2, float64(attempt))
	delayMS := float64(baseDelayMS) * exp
	if delayMS > maxDelayMS {
		delayMS = maxDelayMS
	}

	// Add jitter: random value in [0, baseDelayMS].
	jitterMS := float64(rand.Intn(baseDelayMS))
	totalMS := delayMS + jitterMS

	return time.Duration(totalMS) * time.Millisecond
}
