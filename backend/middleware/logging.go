package middleware

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"runtime/debug"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// statusRecorder wraps ResponseWriter to capture status & size.
type statusRecorder struct {
	http.ResponseWriter
	status int
	size   int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}
func (sr *statusRecorder) Write(b []byte) (int, error) {
	if sr.status == 0 { // implicit 200
		sr.status = http.StatusOK
	}
	n, err := sr.ResponseWriter.Write(b)
	sr.size += n
	return n, err
}

// Hijack delegates to the underlying ResponseWriter if it supports http.Hijacker.
// This is required for WebSocket upgrades to function when wrapped by logging middleware.
func (sr *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := sr.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, fmt.Errorf("underlying ResponseWriter does not support hijacking")
}

// Flush delegates to the underlying ResponseWriter if it supports http.Flusher.
func (sr *statusRecorder) Flush() {
	if f, ok := sr.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Push delegates HTTP/2 server push if supported; ignored otherwise.
func (sr *statusRecorder) Push(target string, opts *http.PushOptions) error {
	if p, ok := sr.ResponseWriter.(http.Pusher); ok {
		return p.Push(target, opts)
	}
	return http.ErrNotSupported
}

var reqIDCounter uint64

// Logging provides basic structured-ish logging with a request id.
// It also recovers from panics, returning 500 and logging stack trace.
func Logging(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rid := fmt.Sprintf("%d-%x", atomic.AddUint64(&reqIDCounter, 1), start.UnixNano())
			w.Header().Set("X-Request-ID", rid)
			sr := &statusRecorder{ResponseWriter: w}
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("panic",
						zap.String("request_id", rid),
						zap.String("method", r.Method),
						zap.String("path", r.URL.Path),
						zap.Any("error", rec),
						zap.ByteString("stack", debug.Stack()),
					)
					http.Error(sr, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
				dur := time.Since(start)
				logger.Info("request",
					zap.String("request_id", rid),
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
					zap.Int("status", sr.status),
					zap.Int("bytes", sr.size),
					zap.Int64("duration_ms", dur.Milliseconds()),
				)
			}()
			next.ServeHTTP(sr, r)
		})
	}
}
