package bot

import (
	"context"
	"encoding/json"
	"english-bot/internal/database"
	"english-bot/internal/services"
	"fmt"
	"log/slog"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// State представляет состояние пользователя
const (
	StateIdle          = "idle"
	StateChat          = "chat"
	StateGrammarCheck  = "grammar_check"
	StateExercise      = "exercise"
	StateExerciseReply = "exercise_reply"
)

// Handler обрабатывает сообщения от пользователей
type Handler struct {
	bot             *tgbotapi.BotAPI
	db              *database.PostgresDB
	openAI          *services.OpenAIService
	languageTool    *services.LanguageToolService
	exerciseService *services.ExerciseService
	progressService *services.ProgressService
}

// NewHandler создает новый обработчик сообщений
func NewHandler(bot *tgbotapi.BotAPI, db *database.PostgresDB, openAI *services.OpenAIService) *Handler {
	return &Handler{
		bot:    bot,
		db:     db,
		openAI: openAI,
	}
}

// SetLanguageToolService устанавливает сервис LanguageTool
func (h *Handler) SetLanguageToolService(service *services.LanguageToolService) {
	h.languageTool = service
}

// SetExerciseService устанавливает сервис упражнений
func (h *Handler) SetExerciseService(service *services.ExerciseService) {
	h.exerciseService = service
}

// SetProgressService устанавливает сервис прогресса
func (h *Handler) SetProgressService(service *services.ProgressService) {
	h.progressService = service
}

// HandleUpdate обрабатывает обновления от Telegram
func (h *Handler) HandleUpdate(ctx context.Context, update tgbotapi.Update) {
	// Игнорируем обновления без сообщений
	if update.Message == nil {
		return
	}

	slog.Info("Получено сообщение",
		"from", update.Message.From.UserName,
		"text", update.Message.Text,
	)

	// Получаем или создаем пользователя в БД
	user, err := h.getOrCreateUser(ctx, update.Message.From)
	if err != nil {
		slog.Error("Ошибка получения пользователя", "error", err)
		h.sendErrorMessage(update.Message.Chat.ID)
		return
	}

	// Получаем текущую сессию пользователя
	session, err := h.db.GetOrCreateUserSession(ctx, user.ID)
	if err != nil {
		slog.Error("Ошибка получения сессии", "error", err)
		h.sendErrorMessage(update.Message.Chat.ID)
		return
	}

	// Обрабатываем команды
	if update.Message.IsCommand() {
		h.handleCommand(ctx, update, user, session)
		return
	}

	// Обрабатываем сообщения в зависимости от состояния
	h.handleMessageByState(ctx, update, user, session)
}

// getOrCreateUser получает или создает пользователя в БД
func (h *Handler) getOrCreateUser(ctx context.Context, tgUser *tgbotapi.User) (*database.User, error) {
	user, err := h.db.GetUserByTelegramID(ctx, tgUser.ID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения пользователя: %w", err)
	}

	if user == nil {
		// Создаем нового пользователя
		newUser := database.User{
			TelegramID:   tgUser.ID,
			Username:     tgUser.UserName,
			FirstName:    tgUser.FirstName,
			LastName:     tgUser.LastName,
			LanguageCode: tgUser.LanguageCode,
			EnglishLevel: "A1", // По умолчанию A1
		}

		user, err = h.db.CreateUser(ctx, newUser)
		if err != nil {
			return nil, fmt.Errorf("ошибка создания пользователя: %w", err)
		}

		// Обновляем статистику для нового пользователя
		h.db.UpdateUserStreak(ctx, user.ID)
	}

	return user, nil
}

// handleCommand обрабатывает команды бота
func (h *Handler) handleCommand(ctx context.Context, update tgbotapi.Update, user *database.User, session *database.UserSession) {
	chatID := update.Message.Chat.ID
	command := update.Message.Command()

	switch command {
	case "start":
		// Сбрасываем состояние на idle
		session.State = StateIdle
		h.db.UpdateUserSession(ctx, *session)

		msg := tgbotapi.NewMessage(chatID,
			"👋 *Welcome to English Learning Bot!*\n\n"+
				"I'm here to help you learn English in an interactive and fun way. You can:\n"+
				"• Chat with me in English\n"+
				"• Check your grammar\n"+
				"• Get personalized exercises\n"+
				"• Track your progress\n\n"+
				"Use /help to see all available commands.")
		msg.ParseMode = "Markdown"
		h.bot.Send(msg)

	case "help":
		msg := tgbotapi.NewMessage(chatID,
			"*Available commands:*\n\n"+
				"📝 */chat* - Start a conversation in English\n"+
				"✅ */check* - Check grammar of your sentence\n"+
				"📚 */exercise* - Get a new exercise\n"+
				"📊 */progress* - Show your learning progress\n"+
				"⚙️ */settings* - Change your preferences")
		msg.ParseMode = "Markdown"
		h.bot.Send(msg)

	case "chat":
		// Устанавливаем состояние чата
		session.State = StateChat
		h.db.UpdateUserSession(ctx, *session)

		// Начинаем новый диалог
		conversation, err := h.db.StartConversation(ctx, user.ID, "general", user.EnglishLevel)
		if err != nil {
			slog.Error("Ошибка создания диалога", "error", err)
			h.sendErrorMessage(chatID)
			return
		}

		// Сохраняем ID диалога в сессии
		session.ConversationID = fmt.Sprintf("%d", conversation.ID)
		h.db.UpdateUserSession(ctx, *session)

		msg := tgbotapi.NewMessage(chatID,
			"🗣️ *Let's practice English!*\n\n"+
				"I'll be your conversation partner. Feel free to talk about anything you want.\n"+
				"Just type your message in English, and I'll respond.")
		msg.ParseMode = "Markdown"
		h.bot.Send(msg)

	case "check":
		// Устанавливаем состояние проверки грамматики
		session.State = StateGrammarCheck
		h.db.UpdateUserSession(ctx, *session)

		msg := tgbotapi.NewMessage(chatID,
			"✅ *Grammar Check Mode*\n\n"+
				"Send me a sentence or paragraph in English, and I'll check it for grammar mistakes.\n"+
				"I'll explain any errors and suggest corrections.")
		msg.ParseMode = "Markdown"
		h.bot.Send(msg)

	case "exercise":
		// Устанавливаем состояние упражнения
		session.State = StateExercise

		// Сохраняем в контексте тип упражнения (пока генерируем базовое)
		contextData := map[string]string{
			"exerciseType": "grammar",
		}

		contextJSON, _ := json.Marshal(contextData)
		session.ContextData = contextJSON
		h.db.UpdateUserSession(ctx, *session)

		// Отправляем сообщение о генерации упражнения
		msg := tgbotapi.NewMessage(chatID, "🔄 Generating exercise for your level, please wait...")
		waitMsg, _ := h.bot.Send(msg)

		// Генерируем упражнение через OpenAI
		exerciseText, err := h.openAI.GenerateExercise("grammar", user.EnglishLevel)
		if err != nil {
			slog.Error("Ошибка генерации упражнения", "error", err)
			h.sendErrorMessage(chatID)
			return
		}

		// Сохраняем упражнение в БД
		exercise := database.Exercise{
			Type:    "grammar",
			Level:   user.EnglishLevel,
			Content: exerciseText,
			// Ответ будет заполнен позже для упражнений с ожидаемым ответом
		}

		savedExercise, err := h.db.SaveExercise(ctx, exercise)
		if err != nil {
			slog.Error("Ошибка сохранения упражнения", "error", err)
		}

		// Обновляем контекст сессии, включая ID упражнения
		contextData["exerciseID"] = fmt.Sprintf("%d", savedExercise.ID)
		contextJSON, _ = json.Marshal(contextData)
		session.ContextData = contextJSON
		session.State = StateExerciseReply
		h.db.UpdateUserSession(ctx, *session)

		// Удаляем сообщение "Генерируем упражнение"
		deleteMsg := tgbotapi.NewDeleteMessage(chatID, waitMsg.MessageID)
		h.bot.Request(deleteMsg)

		// Отправляем упражнение
		exerciseMsg := tgbotapi.NewMessage(chatID,
			"📚 *Exercise*\n\n"+exerciseText+"\n\n"+
				"Type your answer when ready.")
		exerciseMsg.ParseMode = "Markdown"
		h.bot.Send(exerciseMsg)

	case "progress":
		progress, err := h.db.GetUserProgress(ctx, user.ID)
		if err != nil {
			slog.Error("Ошибка получения прогресса", "error", err)
			h.sendErrorMessage(chatID)
			return
		}

		// Рассчитываем процент правильных ответов
		correctPercentage := 0
		if progress.TotalExercises > 0 {
			correctPercentage = (progress.CorrectExercises * 100) / progress.TotalExercises
		}

		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(
			"📊 *Your Learning Progress*\n\n"+
				"• English Level: *%s*\n"+
				"• Exercises Completed: *%d*\n"+
				"• Correct Answers: *%d (%d%%)*\n"+
				"• Conversations: *%d*\n"+
				"• Messages Exchanged: *%d*\n"+
				"• Current Streak: *%d days*\n"+
				"• Longest Streak: *%d days*\n\n"+
				"Keep up the good work! 🌟",
			user.EnglishLevel,
			progress.TotalExercises,
			progress.CorrectExercises,
			correctPercentage,
			progress.TotalConversations,
			progress.TotalMessages,
			progress.CurrentStreak,
			progress.LongestStreak,
		))
		msg.ParseMode = "Markdown"
		h.bot.Send(msg)

	default:
		msg := tgbotapi.NewMessage(chatID, "Unknown command. Use /help to see available commands.")
		h.bot.Send(msg)
	}
}

// handleMessageByState обрабатывает сообщения в зависимости от состояния
func (h *Handler) handleMessageByState(ctx context.Context, update tgbotapi.Update, user *database.User, session *database.UserSession) {
	chatID := update.Message.Chat.ID
	text := update.Message.Text

	switch session.State {
	case StateChat:
		// Получаем ID беседы из сессии
		conversationID := 0
		fmt.Sscanf(session.ConversationID, "%d", &conversationID)

		if conversationID == 0 {
			// Если ID беседы не найден, создаем новую
			conversation, err := h.db.StartConversation(ctx, user.ID, "general", user.EnglishLevel)
			if err != nil {
				slog.Error("Ошибка создания диалога", "error", err)
				h.sendErrorMessage(chatID)
				return
			}
			conversationID = int(conversation.ID)
			session.ConversationID = fmt.Sprintf("%d", conversationID)
			h.db.UpdateUserSession(ctx, *session)
		}

		// Сохраняем сообщение пользователя
		userMessage := database.ConversationMessage{
			ConversationID: int64(conversationID),
			Role:           "user",
			Content:        text,
		}
		h.db.AddConversationMessage(ctx, userMessage)

		// Отправляем сообщение о печатании
		typingMsg := tgbotapi.NewChatAction(chatID, tgbotapi.ChatTyping)
		h.bot.Request(typingMsg)

		// Создаем системный промпт в зависимости от уровня пользователя
		systemPrompt := fmt.Sprintf("You are an English tutor speaking with a student at %s level. Be encouraging, correct major mistakes, and adapt your language to their level. Keep responses concise and natural. Respond in English only.", user.EnglishLevel)

		// Получаем ответ от OpenAI
		response, err := h.openAI.GenerateResponse(text, systemPrompt)
		if err != nil {
			slog.Error("Ошибка получения ответа от OpenAI", "error", err)
			h.sendErrorMessage(chatID)
			return
		}

		// Сохраняем ответ бота
		botMessage := database.ConversationMessage{
			ConversationID: int64(conversationID),
			Role:           "bot",
			Content:        response,
		}
		h.db.AddConversationMessage(ctx, botMessage)

		// Отправляем ответ пользователю
		msg := tgbotapi.NewMessage(chatID, response)
		h.bot.Send(msg)

		// Обновляем статистику пользователя
		h.db.UpdateUserStreak(ctx, user.ID)

	case StateGrammarCheck:
		// Отправляем сообщение о проверке
		waitMsg := tgbotapi.NewMessage(chatID, "🔍 Checking grammar...")
		sentMsg, _ := h.bot.Send(waitMsg)

		// Проверяем грамматику через OpenAI
		result, err := h.openAI.CheckGrammar(text)
		if err != nil {
			slog.Error("Ошибка проверки грамматики", "error", err)
			h.sendErrorMessage(chatID)
			return
		}

		// Удаляем сообщение "Проверяем грамматику"
		deleteMsg := tgbotapi.NewDeleteMessage(chatID, sentMsg.MessageID)
		h.bot.Request(deleteMsg)

		// Отправляем результат проверки
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("✅ *Grammar Check Result*\n\n%s", result))
		msg.ParseMode = "Markdown"
		h.bot.Send(msg)

		// Сбрасываем состояние
		session.State = StateIdle
		h.db.UpdateUserSession(ctx, *session)

		// Обновляем статистику пользователя
		h.db.UpdateUserStreak(ctx, user.ID)

	case StateExerciseReply:
		// Получаем данные контекста
		var contextData map[string]string
		if err := json.Unmarshal(session.ContextData, &contextData); err != nil {
			slog.Error("Ошибка разбора контекста сессии", "error", err)
			h.sendErrorMessage(chatID)
			return
		}

		exerciseID := 0
		fmt.Sscanf(contextData["exerciseID"], "%d", &exerciseID)

		// Отправляем сообщение о проверке ответа
		waitMsg := tgbotapi.NewMessage(chatID, "🔍 Checking your answer...")
		sentMsg, _ := h.bot.Send(waitMsg)

		// Здесь будет проверка ответа через OpenAI
		// Для демонстрации просто используем базовую проверку
		isCorrect := strings.Contains(strings.ToLower(text), "correct") // это просто заглушка

		// Сохраняем ответ пользователя
		userExercise := database.UserExercise{
			UserID:     user.ID,
			ExerciseID: int64(exerciseID),
			UserAnswer: text,
			IsCorrect:  isCorrect,
		}
		h.db.SaveUserExercise(ctx, userExercise)

		// Удаляем сообщение "Проверяем ответ"
		deleteMsg := tgbotapi.NewDeleteMessage(chatID, sentMsg.MessageID)
		h.bot.Request(deleteMsg)

		var feedbackMsg string
		if isCorrect {
			feedbackMsg = "🎉 *Correct!*\n\nGreat job! Your answer is correct."
		} else {
			feedbackMsg = "❌ *Not quite right*\n\nLet me explain:\n\n" +
				"Your answer has some issues. Here's a better way to answer:\n" +
				"[Правильный ответ будет здесь]"
		}

		// Отправляем результат
		msg := tgbotapi.NewMessage(chatID, feedbackMsg)
		msg.ParseMode = "Markdown"
		h.bot.Send(msg)

		// Предлагаем следующее упражнение
		nextMsg := tgbotapi.NewMessage(chatID, "Would you like another exercise? Use /exercise to get one.")
		h.bot.Send(nextMsg)

		// Сбрасываем состояние
		session.State = StateIdle
		h.db.UpdateUserSession(ctx, *session)

		// Обновляем статистику пользователя
		h.db.UpdateUserStreak(ctx, user.ID)

	default:
		// Для неизвестного состояния предлагаем команды
		msg := tgbotapi.NewMessage(chatID,
			"I'm not sure what you want to do. Here are some options:\n\n"+
				"• Use /chat to practice speaking English\n"+
				"• Use /check to check grammar\n"+
				"• Use /exercise to get a learning exercise\n"+
				"• Use /progress to see your stats")
		h.bot.Send(msg)

		// Сбрасываем состояние
		session.State = StateIdle
		h.db.UpdateUserSession(ctx, *session)
	}
}

// sendErrorMessage отправляет сообщение об ошибке пользователю
func (h *Handler) sendErrorMessage(chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "Sorry, something went wrong. Please try again later.")
	h.bot.Send(msg)
}
