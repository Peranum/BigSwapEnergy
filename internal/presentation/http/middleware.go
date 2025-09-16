package http

import (
	"sync"
	"time"

	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

type HTTPRateLimitConfig struct {
	RequestsPerMinute int
}

type RateLimitable interface {
	GetRateLimitConfig() HTTPRateLimitConfig
}

type Middleware interface {
	Apply(next fasthttp.RequestHandler) fasthttp.RequestHandler
}

type ClientRateLimit struct {
	requests    int
	lastRequest time.Time
	mutex       sync.RWMutex
}

type RateLimitMiddleware struct {
	config     HTTPRateLimitConfig
	logger     *zap.Logger
	clients    map[string]*ClientRateLimit
	clientsMux sync.RWMutex
}

func NewRateLimitMiddleware(config HTTPRateLimitConfig, logger *zap.Logger) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		config:  config,
		logger:  logger,
		clients: make(map[string]*ClientRateLimit),
	}
}

func (m *RateLimitMiddleware) Apply(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		clientIP := string(ctx.Request.Header.Peek("X-Forwarded-For"))
		if clientIP == "" {
			clientIP = ctx.RemoteIP().String()
		}

		if !m.checkRateLimit(clientIP) {
			m.logger.Warn("Rate limit exceeded",
				zap.String("client_ip", clientIP),
				zap.String("path", string(ctx.Path())),
			)

			ctx.SetStatusCode(fasthttp.StatusTooManyRequests)
			ctx.SetContentType("application/json")
			ctx.SetBodyString(`{"error":{"code":"RATE_LIMIT_EXCEEDED","message":"Rate limit exceeded"}}`)
			return
		}

		next(ctx)
	}
}

func (m *RateLimitMiddleware) checkRateLimit(clientIP string) bool {
	now := time.Now()

	m.clientsMux.Lock()
	defer m.clientsMux.Unlock()

	client, exists := m.clients[clientIP]
	if !exists {
		client = &ClientRateLimit{
			requests:    1,
			lastRequest: now,
		}
		m.clients[clientIP] = client
		return true
	}

	client.mutex.Lock()
	defer client.mutex.Unlock()

	if now.Sub(client.lastRequest) > time.Minute {
		client.requests = 1
		client.lastRequest = now
		return true
	}

	if client.requests >= m.config.RequestsPerMinute {
		return false
	}

	client.requests++
	client.lastRequest = now
	return true
}

func ApplyMiddleware(handler fasthttp.RequestHandler, logger *zap.Logger, configurable interface{}) fasthttp.RequestHandler {
	if rateLimitable, ok := configurable.(RateLimitable); ok {
		rateLimitConfig := rateLimitable.GetRateLimitConfig()
		rateLimitMiddleware := NewRateLimitMiddleware(rateLimitConfig, logger)
		return rateLimitMiddleware.Apply(handler)
	}

	return handler
}
