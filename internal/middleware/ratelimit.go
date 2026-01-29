package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/Igorjr19/go-shorty/internal/logger"
)

type RateLimiter interface {
	Limit(next http.HandlerFunc) http.HandlerFunc
	AllowRequest(identifier string) bool
}

type InMemoryRateLimiter struct {
	visitors map[string]*visitor
	mu       sync.RWMutex
	rate     int
	window   time.Duration
}

type visitor struct {
	requests  int
	lastReset time.Time
	mu        sync.Mutex
}

func NewInMemoryRateLimiter(rate int, window time.Duration) RateLimiter {
	rl := &InMemoryRateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate,
		window:   window,
	}

	go rl.cleanupVisitors()

	return rl
}

func (rl *InMemoryRateLimiter) getVisitor(ip string) *visitor {
	rl.mu.RLock()
	v, exists := rl.visitors[ip]
	rl.mu.RUnlock()

	if exists {
		return v
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	if v, exists := rl.visitors[ip]; exists {
		return v
	}

	v = &visitor{
		lastReset: time.Now(),
		requests:  0,
	}
	rl.visitors[ip] = v
	return v
}

func (rl *InMemoryRateLimiter) cleanupVisitors() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			v.mu.Lock()
			if time.Since(v.lastReset) > rl.window*2 {
				delete(rl.visitors, ip)
			}
			v.mu.Unlock()
		}
		rl.mu.Unlock()
	}
}

func (rl *InMemoryRateLimiter) AllowRequest(identifier string) bool {
	v := rl.getVisitor(identifier)
	v.mu.Lock()
	defer v.mu.Unlock()

	if time.Since(v.lastReset) > rl.window {
		v.requests = 0
		v.lastReset = time.Now()
	}

	if v.requests >= rl.rate {
		return false
	}

	v.requests++
	return true
}

func (rl *InMemoryRateLimiter) Limit(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := getIP(r)

		if !rl.AllowRequest(ip) {
			logger.RateLimitExceeded(r.Context(), ip, rl.rate, rl.window)
			http.Error(w, "Rate limit exceeded. Try again later.", http.StatusTooManyRequests)
			return
		}

		next(w, r)
	}
}

func getIP(r *http.Request) string {
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		ip := forwarded
		for idx := 0; idx < len(forwarded); idx++ {
			if forwarded[idx] == ',' {
				ip = forwarded[:idx]
				break
			}
		}
		return ip
	}

	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
