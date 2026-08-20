package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const csrfHeader = "X-Meander-Request"

var (
	errForbiddenRequest      = errors.New("request origin could not be verified")
	errRateLimited           = errors.New("too many requests; try again later")
	defaultAuthLimiter       = newRateLimiter()
	defaultGenerationLimiter = newRateLimiter()
	defaultGenerationGuard   = newConcurrencyGuard()
)

type rateWindow struct {
	count int
	reset time.Time
}

type rateLimiter struct {
	mu      sync.Mutex
	windows map[string]rateWindow
}

func newRateLimiter() *rateLimiter { return &rateLimiter{windows: map[string]rateWindow{}} }

func (l *rateLimiter) allow(key string, limit int, window time.Duration) (bool, time.Duration) {
	if limit <= 0 {
		return true, 0
	}
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	entry, ok := l.windows[key]
	if !ok || !now.Before(entry.reset) {
		l.windows[key] = rateWindow{count: 1, reset: now.Add(window)}
		return true, 0
	}
	if entry.count >= limit {
		return false, time.Until(entry.reset)
	}
	entry.count++
	l.windows[key] = entry
	return true, 0
}

type concurrencyGuard struct {
	mu     sync.Mutex
	active map[string]int
}

func newConcurrencyGuard() *concurrencyGuard { return &concurrencyGuard{active: map[string]int{}} }

func (g *concurrencyGuard) begin(key string, limit int) (func(), bool) {
	g.mu.Lock()
	if g.active[key] >= limit {
		g.mu.Unlock()
		return nil, false
	}
	g.active[key]++
	g.mu.Unlock()
	return func() {
		g.mu.Lock()
		g.active[key]--
		if g.active[key] <= 0 {
			delete(g.active, key)
		}
		g.mu.Unlock()
	}, true
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Cross-Origin-Resource-Policy", "same-site")
		next.ServeHTTP(w, r)
	})
}

func csrfProtection(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
			fail(w, http.StatusForbidden, errForbiddenRequest)
			return
		}
		origin := r.Header.Get("Origin")
		if origin != "" && !originAllowed(origin) {
			fail(w, http.StatusForbidden, errForbiddenRequest)
			return
		}
		if r.Header.Get(csrfHeader) != "browser" {
			fail(w, http.StatusForbidden, errForbiddenRequest)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	if r.RemoteAddr != "" {
		return r.RemoteAddr
	}
	return "unknown"
}

func rateLimitSubject(value string) string {
	key := []byte(env("MEANDER_RATE_LIMIT_SALT", "local-development-salt"))
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}

func rateLimitFailure(w http.ResponseWriter, retryAfter time.Duration) {
	seconds := max(1, int(retryAfter.Seconds()))
	w.Header().Set("Retry-After", strconv.Itoa(seconds))
	fail(w, http.StatusTooManyRequests, errRateLimited)
}

func positiveEnvInt(key string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(env(key, "")))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
