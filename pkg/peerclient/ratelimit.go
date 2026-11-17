package peerclient

import (
	"sync"
	"time"
)

// rateLimiter : seau à jetons simple (octets/s). nil ou 0 = illimité.
type rateLimiter struct {
	mu     sync.Mutex
	rate   float64
	tokens float64
	last   time.Time
}

func newRateLimiter(bytesPerSec int64) *rateLimiter {
	if bytesPerSec <= 0 {
		return nil
	}
	r := float64(bytesPerSec)
	return &rateLimiter{rate: r, tokens: r, last: time.Now()}
}

// wait bloque jusqu'à ce que n octets puissent partir.
func (l *rateLimiter) wait(n int) {
	if l == nil {
		return
	}
	l.mu.Lock()
	now := time.Now()
	l.tokens = min(l.rate, l.tokens+now.Sub(l.last).Seconds()*l.rate)
	l.last = now
	l.tokens -= float64(n)
	debt := -l.tokens
	l.mu.Unlock()
	if debt > 0 {
		time.Sleep(time.Duration(debt / l.rate * float64(time.Second)))
	}
}
