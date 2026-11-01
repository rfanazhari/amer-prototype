package accesshandler

import (
	"net"
	"sync"
	"time"
)

const (
	loginWindow     = 15 * time.Minute
	emailAttemptMax = 5
	ipAttemptMax    = 20
	retryAfterMax   = 900
)

// LoginLimiter bounds administrative login attempts by canonical email and client IP.
type LoginLimiter struct {
	mu     sync.Mutex
	emails map[string][]time.Time
	ips    map[string][]time.Time
	now    func() time.Time
}

// allow records one login attempt and reports whether the request may proceed.
func (l *LoginLimiter) allow(email, ip string) (bool, int) {
	if l == nil {
		return true, 0
	}
	now := time.Now()
	if l.now != nil {
		now = l.now()
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.emails == nil {
		l.emails = map[string][]time.Time{}
		l.ips = map[string][]time.Time{}
	}
	if retry, blocked := l.exceeded(l.ips, ip, ipAttemptMax, now); blocked {
		return false, retry
	}
	if email != "" {
		if retry, blocked := l.exceeded(l.emails, email, emailAttemptMax, now); blocked {
			return false, retry
		}
		l.emails[email] = append(l.prune(l.emails[email], now), now)
	}
	l.ips[ip] = append(l.prune(l.ips[ip], now), now)
	return true, 0
}

// exceeded reports whether the next attempt would pass the windowed budget.
func (l *LoginLimiter) exceeded(store map[string][]time.Time, key string, max int, now time.Time) (int, bool) {
	stamps := l.prune(store[key], now)
	store[key] = stamps
	if len(stamps) < max {
		return 0, false
	}
	retry := int(stamps[0].Add(loginWindow).Sub(now).Seconds())
	if retry < 1 {
		retry = 1
	}
	if retry > retryAfterMax {
		retry = retryAfterMax
	}
	return retry, true
}

// prune drops timestamps that have left the 15-minute window.
func (l *LoginLimiter) prune(stamps []time.Time, now time.Time) []time.Time {
	cut := now.Add(-loginWindow)
	kept := stamps[:0]
	for _, stamp := range stamps {
		if stamp.After(cut) {
			kept = append(kept, stamp)
		}
	}
	return kept
}

// clientIP extracts the remote host used by the login IP budget.
func clientIP(rAddr string) string {
	host, _, err := net.SplitHostPort(rAddr)
	if err != nil {
		return rAddr
	}
	return host
}
