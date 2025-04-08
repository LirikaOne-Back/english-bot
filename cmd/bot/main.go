package main

import (
	"context"
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
	bot, err := tgbotapi.NewBotAPI(config.TelegramToken)
	if err != nil {
		slog.Error("Ошибка инициализации Telegram API", "error", err)
		os.Exit(1)
	}

	bot.Debug = config.Debug
	slog.Info("Бот успешно авторизован", "username", bot.Self.UserName)

	// Настройка обработки обновлений
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	updates := bot.GetUpdatesChan(updateConfig)

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

	// Обработка сообщений
	go handleUpdates(ctx, bot, updates)

	// Ожидание завершения контекста
	<-ctx.Done()
	slog.Info("Бот остановлен")
}

// Обработка сообщений от пользователей
func handleUpdates(ctx context.Context, bot *tgbotapi.BotAPI, updates tgbotapi.UpdatesChannel) {
	for {
		select {
		case <-ctx.Done():
			return
		case update := <-updates:
			if update.Message == nil {
				continue
			}

			slog.Info("Получено сообщение",
				"from", update.Message.From.UserName,
				"text", update.Message.Text,
			)

			// Обработка команд
			if update.Message.IsCommand() {
				handleCommand(bot, update)
				continue
			}

			// Обработка обычных сообщений
			handleMessage(bot, update)
		}
	}
}

// Обработка команд бота
func handleCommand(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

	switch update.Message.Command() {
	case "start":
		msg.Text = "Привет! Я бот для изучения английского языка. Готов помочь тебе в обучении! Используй /help для списка команд."
	case "help":
		msg.Text = `Доступные команды:
/start - Начать работу с ботом
/help - Показать список команд
/chat - Начать диалог на английском
/check - Проверить грамматику предложения
/exercise - Получить новое упражнение
/progress - Показать ваш прогресс`
	case "chat":
		msg.Text = "Давай начнем диалог на английском! Напиши что-нибудь, и я отвечу."
	case "check":
		msg.Text = "Отправь мне предложение на английском, и я проверю его грамматику."
	case "exercise":
		msg.Text = "Вот твое новое упражнение: [Здесь будет сгенерированное упражнение]"
	case "progress":
		msg.Text = "Твой прогресс: [Здесь будет информация о прогрессе]"
	default:
		msg.Text = "Неизвестная команда. Используй /help для списка доступных команд."
	}

	if _, err := bot.Send(msg); err != nil {
		slog.Error("Ошибка отправки сообщения", "error", err)
	}
}

// Обработка обычных сообщений
func handleMessage(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	// В будущем здесь будет обработка различных типов сообщений
	// В зависимости от контекста взаимодействия с пользователем

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Я получил твое сообщение, но пока не умею на него отвечать сложным образом. Скоро научусь!")

	if _, err := bot.Send(msg); err != nil {
		slog.Error("Ошибка отправки сообщения", "error", err)
	}
}
