package main

import (
	"context"
	"english-bot/internal/bot"
	"english-bot/internal/database"
	"english-bot/internal/services"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

// Конфигурация бота
type Config struct {
	TelegramToken string
	OpenAIToken   string
	DBConnString  string
	Debug         bool
}

// Загрузка конфигурации из .env файла
func loadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("ошибка загрузки .env файла: %w", err)
	}

	return &Config{
		TelegramToken: os.Getenv("TELEGRAM_TOKEN"),
		OpenAIToken:   os.Getenv("OPENAI_TOKEN"),
		DBConnString:  os.Getenv("DATABASE_URL"),
		Debug:         os.Getenv("DEBUG") == "true",
	}, nil
}

// Основная функция запуска бота
func main() {
	// Настройка логгера
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Загрузка конфигурации
	config, err := loadConfig()
	if err != nil {
		slog.Error("Ошибка загрузки конфигурации", "error", err)
		os.Exit(1)
	}

	// Инициализация Telegram бота
	botAPI, err := tgbotapi.NewBotAPI(config.TelegramToken)
	if err != nil {
		slog.Error("Ошибка инициализации Telegram API", "error", err)
		os.Exit(1)
	}

	botAPI.Debug = config.Debug
	slog.Info("Бот успешно авторизован", "username", botAPI.Self.UserName)

	// Подключение к базе данных
	db, err := database.NewPostgresDB(config.DBConnString)
	if err != nil {
		slog.Error("Ошибка подключения к базе данных", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Инициализация сервисов
	openAIService := services.NewOpenAIService(config.OpenAIToken)
	exerciseService := services.NewExerciseService(openAIService)
	languageToolService := services.NewLanguageToolService()
	progressService := services.NewProgressService(db)

	// Инициализация обработчиков
	handler := bot.NewHandler(botAPI, db, openAIService)
	// Установка дополнительных сервисов
	handler.SetExerciseService(exerciseService)
	handler.SetLanguageToolService(languageToolService)
	handler.SetProgressService(progressService)

	middleware := bot.NewMiddleware(*handler)

	// Настройка обработки обновлений
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	updates := botAPI.GetUpdatesChan(updateConfig)

	// Создаем канал для сигналов завершения
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Обработка сигналов остановки
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		slog.Info("Получен сигнал остановки, завершение работы...")
		cancel()
	}()

	// Запуск API сервера на Fiber (опционально)
	app := fiber.New()

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
		})
	})

	go func() {
		if err := app.Listen(":8080"); err != nil {
			slog.Error("Ошибка запуска Fiber сервера", "error", err)
		}
	}()

	// Обработка сообщений через middleware и handler
	go processUpdates(ctx, middleware, updates)

	// Ожидание завершения контекста
	<-ctx.Done()
	slog.Info("Бот остановлен")
}

// processUpdates обрабатывает обновления от Telegram API
func processUpdates(ctx context.Context, middleware *bot.Middleware, updates tgbotapi.UpdatesChannel) {
	for {
		select {
		case <-ctx.Done():
			return
		case update := <-updates:
			// Создаем новый контекст для каждого обновления
			updateCtx := context.Background()

			// Обрабатываем обновление асинхронно
			go middleware.HandleUpdate(updateCtx, update)
		}
	}
}
