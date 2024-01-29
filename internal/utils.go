package internal

import (
	"math/rand"
	"time"
)

type BackoffFunc func(retry int) (sleep time.Duration)

func NewBackoffPolicy(mind, maxd time.Duration) BackoffFunc {
	r := rand.New(rand.NewSource(time.Now().Unix()))

	factor := time.Second
	for _, scale := range []time.Duration{time.Hour, time.Minute, time.Second, time.Millisecond, time.Microsecond, time.Nanosecond} {
		d := mind.Truncate(scale)
		if d > 0 {
			factor = scale
			break
		}
	}

	return func(retry int) (sleep time.Duration) {
		wait := 2 << max(0, min(32, retry)) * factor
		jitter := time.Duration(r.Int63n(max(1, int64(wait)/5))) // max 20% jitter
		wait = mind + wait + jitter
		if wait > maxd {
			wait = maxd
		}
		return wait
	}
}
