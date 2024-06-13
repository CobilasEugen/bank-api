package router

import (
	"net/http"
	"sync"
	"time"
)

type Limiter struct {
	tokens map[string]chan struct{}
	rps    int
	getId  func(*http.Request) string
	mutex  sync.Mutex
}

func NewLimiter(rps int, getId func(*http.Request) string) *Limiter {
	return &Limiter{
		tokens: make(map[string]chan struct{}),
		rps:    rps,
		getId:  getId,
	}
}

func RateLimit(handler http.HandlerFunc, limiter *Limiter) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := limiter.getId(r)

		limiter.mutex.Lock()
		defer limiter.mutex.Unlock()

		// check if id has been seen before
		tokens, ok := limiter.tokens[id]
		if !ok {
			tokens = make(chan struct{}, limiter.rps)
			// fill bucket with tokens
			for i := 0; i < limiter.rps; i++ {
				tokens <- struct{}{}
			}
			limiter.tokens[id] = tokens

			go fillTokenBucket(tokens, limiter.rps)
		}

		select {
		// there are tokens in the bucket
		case <-tokens:
			handler(w, r)
		// the bucket is empty
		default:
			http.Error(w, "Rate Limit Exceeded", http.StatusTooManyRequests)
		}
	})
}

func fillTokenBucket(tokens chan struct{}, rps int) {
	ticker := time.NewTicker(time.Second / time.Duration(rps))
	defer ticker.Stop()

	// add token to the bucket each time Ticker ticks
	for range ticker.C {
		select {
		case tokens <- struct{}{}:
		default:
		}
	}
}
