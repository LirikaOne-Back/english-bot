package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresDB представляет соединение с базой данных PostgreSQL
type PostgresDB struct {
	pool *pgxpool.Pool
}

// NewPostgresDB создает новое соединение с базой данных
func NewPostgresDB(connString string) (*PostgresDB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("ошибка разбора строки подключения: %w", err)
	}

	// Настройка пула соединений
	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnLifetime = 1 * time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к базе данных: %w", err)
	}

	// Проверяем соединение
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ошибка пинга базы данных: %w", err)
	}

	slog.Info("Успешное подключение к базе данных PostgreSQL")

	return &PostgresDB{pool: pool}, nil
}

// Close закрывает соединение с базой данных
func (db *PostgresDB) Close() {
	db.pool.Close()
}

// GetUserByTelegramID находит пользователя по его Telegram ID
func (db *PostgresDB) GetUserByTelegramID(ctx context.Context, telegramID int64) (*User, error) {
	query := `
		SELECT id, telegram_id, username, first_name, last_name, language_code, english_level, created_at, updated_at
		FROM users
		WHERE telegram_id = $1
	`

	var user User
	err := db.pool.QueryRow(ctx, query, telegramID).Scan(
		&user.ID,
		&user.TelegramID,
		&user.Username,
		&user.FirstName,
		&user.LastName,
		&user.LanguageCode,
		&user.EnglishLevel,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // Пользователь не найден
		}
		return nil, fmt.Errorf("ошибка запроса пользователя: %w", err)
	}

	return &user, nil
}

// CreateUser создает нового пользователя
func (db *PostgresDB) CreateUser(ctx context.Context, user User) (*User, error) {
	query := `
		INSERT INTO users (telegram_id, username, first_name, last_name, language_code, english_level, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	// По умолчанию ставим начальный уровень
	if user.EnglishLevel == "" {
		user.EnglishLevel = "A1"
	}

	err := db.pool.QueryRow(ctx, query,
		user.TelegramID,
		user.Username,
		user.FirstName,
		user.LastName,
		user.LanguageCode,
		user.EnglishLevel,
		user.CreatedAt,
		user.UpdatedAt,
	).Scan(&user.ID)

	if err != nil {
		return nil, fmt.Errorf("ошибка создания пользователя: %w", err)
	}

	return &user, nil
}

// GetOrCreateUserSession получает текущую сессию пользователя или создает новую
func (db *PostgresDB) GetOrCreateUserSession(ctx context.Context, userID int64) (*UserSession, error) {
	// Сначала проверяем, есть ли активная сессия
	query := `
		SELECT id, user_id, state, context_data, conversation_id, last_activity, created_at, updated_at
		FROM user_sessions
		WHERE user_id = $1
	`

	var session UserSession
	err := db.pool.QueryRow(ctx, query, userID).Scan(
		&session.ID,
		&session.UserID,
		&session.State,
		&session.ContextData,
		&session.ConversationID,
		&session.LastActivity,
		&session.CreatedAt,
		&session.UpdatedAt,
	)

	if err == nil {
		// Сессия найдена, обновляем время последней активности
		updateQuery := `
			UPDATE user_sessions
			SET last_activity = $1, updated_at = $1
			WHERE id = $2
		`
		now := time.Now()
		_, err = db.pool.Exec(ctx, updateQuery, now, session.ID)
		if err != nil {
			slog.Error("Ошибка обновления времени активности сессии", "error", err)
		}
		return &session, nil
	}

	if err != pgx.ErrNoRows {
		return nil, fmt.Errorf("ошибка запроса сессии пользователя: %w", err)
	}

	// Создаем новую сессию
	createQuery := `
		INSERT INTO user_sessions (user_id, state, context_data, last_activity, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $4, $4)
		RETURNING id, created_at, updated_at
	`

	now := time.Now()
	session = UserSession{
		UserID:       userID,
		State:        "idle", // Начальное состояние
		ContextData:  []byte("{}"),
		LastActivity: now,
	}

	err = db.pool.QueryRow(ctx, createQuery,
		session.UserID,
		session.State,
		session.ContextData,
		now,
	).Scan(
		&session.ID,
		&session.CreatedAt,
		&session.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("ошибка создания сессии пользователя: %w", err)
	}

	return &session, nil
}

// UpdateUserSession обновляет сессию пользователя
func (db *PostgresDB) UpdateUserSession(ctx context.Context, session UserSession) error {
	query := `
		UPDATE user_sessions
		SET state = $1, context_data = $2, conversation_id = $3, last_activity = $4, updated_at = $4
		WHERE id = $5
	`

	now := time.Now()
	_, err := db.pool.Exec(ctx, query,
		session.State,
		session.ContextData,
		session.ConversationID,
		now,
		session.ID,
	)

	if err != nil {
		return fmt.Errorf("ошибка обновления сессии пользователя: %w", err)
	}

	return nil
}

// SaveExercise сохраняет новое упражнение
func (db *PostgresDB) SaveExercise(ctx context.Context, exercise Exercise) (*Exercise, error) {
	query := `
		INSERT INTO exercises (type, level, content, answer, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	now := time.Now()
	exercise.CreatedAt = now

	err := db.pool.QueryRow(ctx, query,
		exercise.Type,
		exercise.Level,
		exercise.Content,
		exercise.Answer,
		exercise.CreatedAt,
	).Scan(&exercise.ID)

	if err != nil {
		return nil, fmt.Errorf("ошибка сохранения упражнения: %w", err)
	}

	return &exercise, nil
}

// SaveUserExercise сохраняет ответ пользователя на упражнение
func (db *PostgresDB) SaveUserExercise(ctx context.Context, userExercise UserExercise) (*UserExercise, error) {
	query := `
		INSERT INTO user_exercises (user_id, exercise_id, user_answer, is_correct, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	now := time.Now()
	userExercise.CreatedAt = now

	err := db.pool.QueryRow(ctx, query,
		userExercise.UserID,
		userExercise.ExerciseID,
		userExercise.UserAnswer,
		userExercise.IsCorrect,
		userExercise.CreatedAt,
	).Scan(&userExercise.ID)

	if err != nil {
		return nil, fmt.Errorf("ошибка сохранения ответа на упражнение: %w", err)
	}

	// Обновляем статистику пользователя
	updateQuery := `
		UPDATE user_progress
		SET 
			total_exercises = total_exercises + 1,
			correct_exercises = correct_exercises + CASE WHEN $1 THEN 1 ELSE 0 END,
			updated_at = $2
		WHERE user_id = $3
	`

	_, err = db.pool.Exec(ctx, updateQuery,
		userExercise.IsCorrect,
		now,
		userExercise.UserID,
	)

	if err != nil {
		slog.Error("Ошибка обновления прогресса пользователя", "error", err)
	}

	return &userExercise, nil
}

// StartConversation начинает новый диалог
func (db *PostgresDB) StartConversation(ctx context.Context, userID int64, topic string, level string) (*Conversation, error) {
	query := `
		INSERT INTO conversations (user_id, topic, level, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $4)
		RETURNING id, created_at
	`

	now := time.Now()
	var conversation Conversation
	conversation.UserID = userID
	conversation.Topic = topic
	conversation.Level = level

	err := db.pool.QueryRow(ctx, query,
		conversation.UserID,
		conversation.Topic,
		conversation.Level,
		now,
	).Scan(
		&conversation.ID,
		&conversation.CreatedAt,
	)
	conversation.UpdatedAt = conversation.CreatedAt

	if err != nil {
		return nil, fmt.Errorf("ошибка создания диалога: %w", err)
	}

	return &conversation, nil
}

// AddConversationMessage добавляет сообщение в диалог
func (db *PostgresDB) AddConversationMessage(ctx context.Context, message ConversationMessage) (*ConversationMessage, error) {
	query := `
		INSERT INTO conversation_messages (conversation_id, role, content, created_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	now := time.Now()
	message.CreatedAt = now

	err := db.pool.QueryRow(ctx, query,
		message.ConversationID,
		message.Role,
		message.Content,
		message.CreatedAt,
	).Scan(&message.ID)

	if err != nil {
		return nil, fmt.Errorf("ошибка сохранения сообщения диалога: %w", err)
	}

	// Обновляем время последнего обновления диалога
	updateQuery := `
		UPDATE conversations
		SET updated_at = $1
		WHERE id = $2
	`

	_, err = db.pool.Exec(ctx, updateQuery, now, message.ConversationID)
	if err != nil {
		slog.Error("Ошибка обновления времени диалога", "error", err)
	}

	// Обновляем статистику пользователя
	updateProgressQuery := `
		UPDATE user_progress
		SET total_messages = total_messages + 1,
		    updated_at = $1
		WHERE user_id = (
			SELECT user_id FROM conversations WHERE id = $2
		)
	`

	_, err = db.pool.Exec(ctx, updateProgressQuery, now, message.ConversationID)
	if err != nil {
		slog.Error("Ошибка обновления статистики сообщений пользователя", "error", err)
	}

	return &message, nil
}

// GetUserProgress получает прогресс пользователя
func (db *PostgresDB) GetUserProgress(ctx context.Context, userID int64) (*UserProgress, error) {
	query := `
		SELECT id, user_id, total_exercises, correct_exercises, total_conversations, 
		       total_messages, grammar_corrections, current_streak, longest_streak, 
		       last_activity_date, created_at, updated_at
		FROM user_progress
		WHERE user_id = $1
	`

	var progress UserProgress
	err := db.pool.QueryRow(ctx, query, userID).Scan(
		&progress.ID,
		&progress.UserID,
		&progress.TotalExercises,
		&progress.CorrectExercises,
		&progress.TotalConversations,
		&progress.TotalMessages,
		&progress.GrammarCorrections,
		&progress.CurrentStreak,
		&progress.LongestStreak,
		&progress.LastActivityDate,
		&progress.CreatedAt,
		&progress.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			// Создаем запись о прогрессе, если ее еще нет
			return db.CreateUserProgress(ctx, userID)
		}
		return nil, fmt.Errorf("ошибка получения прогресса пользователя: %w", err)
	}

	return &progress, nil
}

// CreateUserProgress создает запись прогресса для нового пользователя
func (db *PostgresDB) CreateUserProgress(ctx context.Context, userID int64) (*UserProgress, error) {
	query := `
		INSERT INTO user_progress (
			user_id, total_exercises, correct_exercises, total_conversations, 
			total_messages, grammar_corrections, current_streak, longest_streak, 
			last_activity_date, created_at, updated_at
		)
		VALUES ($1, 0, 0, 0, 0, 0, 0, 0, $2, $2, $2)
		RETURNING id, created_at, updated_at
	`

	now := time.Now()
	progress := UserProgress{
		UserID:           userID,
		LastActivityDate: now,
	}

	err := db.pool.QueryRow(ctx, query,
		progress.UserID,
		now,
	).Scan(
		&progress.ID,
		&progress.CreatedAt,
		&progress.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("ошибка создания записи прогресса пользователя: %w", err)
	}

	return &progress, nil
}

// UpdateUserStreak обновляет серии дней активности пользователя
func (db *PostgresDB) UpdateUserStreak(ctx context.Context, userID int64) error {
	// Получаем текущий прогресс
	progress, err := db.GetUserProgress(ctx, userID)
	if err != nil {
		return fmt.Errorf("ошибка получения прогресса для обновления серии: %w", err)
	}

	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)

	// Проверяем, был ли пользователь активен вчера
	if progress.LastActivityDate.Year() == yesterday.Year() &&
		progress.LastActivityDate.Month() == yesterday.Month() &&
		progress.LastActivityDate.Day() == yesterday.Day() {
		// Увеличиваем серию на 1
		progress.CurrentStreak++
	} else if progress.LastActivityDate.Year() == now.Year() &&
		progress.LastActivityDate.Month() == now.Month() &&
		progress.LastActivityDate.Day() == now.Day() {
		// Уже активен сегодня, ничего не делаем
		return nil
	} else {
		// Серия прервалась, начинаем новую
		progress.CurrentStreak = 1
	}

	// Обновляем самую длинную серию, если нужно
	if progress.CurrentStreak > progress.LongestStreak {
		progress.LongestStreak = progress.CurrentStreak
	}

	// Обновляем дату последней активности
	progress.LastActivityDate = now
	progress.UpdatedAt = now

	// Обновляем запись в БД
	updateQuery := `
		UPDATE user_progress
		SET current_streak = $1, longest_streak = $2, 
		    last_activity_date = $3, updated_at = $3
		WHERE id = $4
	`

	_, err = db.pool.Exec(ctx, updateQuery,
		progress.CurrentStreak,
		progress.LongestStreak,
		now,
		progress.ID,
	)

	if err != nil {
		return fmt.Errorf("ошибка обновления серии пользователя: %w", err)
	}

	// Проверяем, есть ли новые достижения
	if progress.CurrentStreak == 7 || progress.CurrentStreak == 30 || progress.CurrentStreak == 100 {
		achievementType := fmt.Sprintf("streak_%d_days", progress.CurrentStreak)
		title := fmt.Sprintf("Серия %d дней", progress.CurrentStreak)
		description := fmt.Sprintf("Вы занимались английским %d дней подряд!", progress.CurrentStreak)

		db.AddUserAchievement(ctx, userID, achievementType, title, description)
	}

	return nil
}

// AddUserAchievement добавляет достижение пользователю
func (db *PostgresDB) AddUserAchievement(ctx context.Context, userID int64, achievementType, title, description string) error {
	// Сначала проверяем, есть ли уже такое достижение
	checkQuery := `
		SELECT id FROM user_achievements
		WHERE user_id = $1 AND achievement_type = $2
	`

	var id int64
	err := db.pool.QueryRow(ctx, checkQuery, userID, achievementType).Scan(&id)
	if err == nil {
		// Достижение уже есть, ничего не делаем
		return nil
	}

	if err != pgx.ErrNoRows {
		return fmt.Errorf("ошибка проверки достижения: %w", err)
	}

	// Добавляем новое достижение
	insertQuery := `
		INSERT INTO user_achievements (user_id, achievement_type, title, description, unlocked_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	now := time.Now()
	_, err = db.pool.Exec(ctx, insertQuery,
		userID,
		achievementType,
		title,
		description,
		now,
	)

	if err != nil {
		return fmt.Errorf("ошибка добавления достижения: %w", err)
	}

	return nil
}
