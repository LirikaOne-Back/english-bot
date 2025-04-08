package bot

import (
	"context"
	"log/slog"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Middleware предоставляет функциональность для обработки обновлений перед их обработкой основными обработчиками
type Middleware struct {
	next Handler
}

// NewMiddleware создает новый middleware
func NewMiddleware(next Handler) *Middleware {
	return &Middleware{
		next: next,
	}
}

// HandleUpdate обрабатывает обновления с применением middleware
func (m *Middleware) HandleUpdate(ctx context.Context, update tgbotapi.Update) {
	// Начало обработки запроса
	startTime := time.Now()

	// Логирование запроса
	if update.Message != nil {
		slog.Info("Incoming message",
			"chat_id", update.Message.Chat.ID,
			"user_id", update.Message.From.ID,
			"username", update.Message.From.UserName,
			"command", update.Message.Command(),
			"text_length", len(update.Message.Text),
		)
	}

	// Создаем контекст с таймаутом для обработки сообщения
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Передаем обновление следующему обработчику
	m.next.HandleUpdate(ctx, update)

	// Логирование времени обработки
	duration := time.Since(startTime)
	slog.Debug("Request processed",
		"duration_ms", duration.Milliseconds(),
	)
}

// RateLimiter ограничивает количество запросов от одного пользователя
// Это заготовка для будущей реализации
type RateLimiter struct {
	next   Handler
	limits map[int64]time.Time // user_id -> last_request_time
}

// NewRateLimiter создает новый ограничитель запросов
func NewRateLimiter(next Handler) *RateLimiter {
	return &RateLimiter{
		next:   next,
		limits: make(map[int64]time.Time),
	}
}

// HandleUpdate обрабатывает обновления с применением ограничения запросов
func (r *RateLimiter) HandleUpdate(ctx context.Context, update tgbotapi.Update) {
	// Проверяем, есть ли сообщение
	if update.Message == nil {
		r.next.HandleUpdate(ctx, update)
		return
	}

	userID := update.Message.From.ID
	now := time.Now()

	// Проверяем последний запрос пользователя
	lastRequest, ok := r.limits[userID]
	if ok {
		// Если последний запрос был менее 1 секунды назад, ограничиваем
		if now.Sub(lastRequest) < 1*time.Second {
			slog.Warn("Rate limit exceeded", "user_id", userID)
			// Здесь можно отправить сообщение пользователю
			return
		}
	}

	// Обновляем время последнего запроса
	r.limits[userID] = now

	// Передаем обновление следующему обработчику
	r.next.HandleUpdate(ctx, update)
}
