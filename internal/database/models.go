package database

import (
	"time"
)

// User представляет пользователя бота
type User struct {
	ID           int64     `db:"id"`
	TelegramID   int64     `db:"telegram_id"`
	Username     string    `db:"username"`
	FirstName    string    `db:"first_name"`
	LastName     string    `db:"last_name"`
	LanguageCode string    `db:"language_code"`
	EnglishLevel string    `db:"english_level"` // A1, A2, B1, B2, C1, C2
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

// UserSession хранит текущее состояние взаимодействия с пользователем
type UserSession struct {
	ID             int64     `db:"id"`
	UserID         int64     `db:"user_id"`
	State          string    `db:"state"`           // chat, exercise, grammar_check и т.д.
	ContextData    []byte    `db:"context_data"`    // JSON с контекстными данными
	ConversationID string    `db:"conversation_id"` // ID текущего разговора
	LastActivity   time.Time `db:"last_activity"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

// Exercise представляет упражнение
type Exercise struct {
	ID        int64     `db:"id"`
	Type      string    `db:"type"`    // grammar, vocabulary, translation и т.д.
	Level     string    `db:"level"`   // A1, A2, B1, B2, C1, C2
	Content   string    `db:"content"` // Содержание упражнения
	Answer    string    `db:"answer"`  // Правильный ответ или ключ
	CreatedAt time.Time `db:"created_at"`
}

// UserExercise связывает пользователя с упражнением
type UserExercise struct {
	ID         int64     `db:"id"`
	UserID     int64     `db:"user_id"`
	ExerciseID int64     `db:"exercise_id"`
	UserAnswer string    `db:"user_answer"` // Ответ пользователя
	IsCorrect  bool      `db:"is_correct"`
	CreatedAt  time.Time `db:"created_at"`
}

// Conversation представляет диалог на английском
type Conversation struct {
	ID        int64     `db:"id"`
	UserID    int64     `db:"user_id"`
	Topic     string    `db:"topic"` // Тема диалога
	Level     string    `db:"level"` // Уровень сложности
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// ConversationMessage представляет сообщение в диалоге
type ConversationMessage struct {
	ID             int64     `db:"id"`
	ConversationID int64     `db:"conversation_id"`
	Role           string    `db:"role"`    // user или bot
	Content        string    `db:"content"` // Текст сообщения
	CreatedAt      time.Time `db:"created_at"`
}

// UserProgress хранит данные о прогрессе пользователя
type UserProgress struct {
	ID                 int64     `db:"id"`
	UserID             int64     `db:"user_id"`
	TotalExercises     int       `db:"total_exercises"`
	CorrectExercises   int       `db:"correct_exercises"`
	TotalConversations int       `db:"total_conversations"`
	TotalMessages      int       `db:"total_messages"`
	GrammarCorrections int       `db:"grammar_corrections"`
	CurrentStreak      int       `db:"current_streak"` // Текущая серия дней занятий
	LongestStreak      int       `db:"longest_streak"` // Самая длинная серия
	LastActivityDate   time.Time `db:"last_activity_date"`
	CreatedAt          time.Time `db:"created_at"`
	UpdatedAt          time.Time `db:"updated_at"`
}

// UserAchievement представляет достижение пользователя
type UserAchievement struct {
	ID              int64     `db:"id"`
	UserID          int64     `db:"user_id"`
	AchievementType string    `db:"achievement_type"` // streak_7_days, exercises_100 и т.д.
	Title           string    `db:"title"`
	Description     string    `db:"description"`
	UnlockedAt      time.Time `db:"unlocked_at"`
}

// UserVocabulary хранит словарь пользователя
type UserVocabulary struct {
	ID          int64     `db:"id"`
	UserID      int64     `db:"user_id"`
	Word        string    `db:"word"`
	Translation string    `db:"translation"`
	Examples    string    `db:"examples"`
	Mastery     int       `db:"mastery"` // 0-5, степень усвоения слова
	LastReview  time.Time `db:"last_review"`
	NextReview  time.Time `db:"next_review"` // Дата следующего повторения
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}
