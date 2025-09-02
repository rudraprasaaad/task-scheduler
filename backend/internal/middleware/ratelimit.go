package middleware

import (
	"net"
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

func RateLimitMiddleware(requestPerSecond int, burst int) func(http.Handler) http.Handler {
	visitors := make(map[string]*rate.Limiter)
	var mu sync.Mutex

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, _ := net.SplitHostPort(r.RemoteAddr)

			mu.Lock()
			limiter, exists := visitors[ip]
			if !exists {
				limiter = rate.NewLimiter(rate.Limit(requestPerSecond), burst)
				visitors[ip] = limiter
			}
			mu.Unlock()

			if !limiter.Allow() {
				http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
