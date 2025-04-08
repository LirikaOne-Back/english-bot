package services

import (
	"context"
	"english-bot/internal/database"
	"fmt"
	"math"
	"time"
)

// ProgressService предоставляет функциональность для работы с прогрессом пользователя
type ProgressService struct {
	db *database.PostgresDB
}

// UserStats представляет статистику пользователя
type UserStats struct {
	TotalExercises       int       // Общее количество упражнений
	CorrectExercises     int       // Правильно выполненные упражнения
	SuccessRate          float64   // Процент успешных упражнений
	TotalConversations   int       // Общее количество диалогов
	TotalMessages        int       // Общее количество сообщений
	CurrentStreak        int       // Текущая серия дней
	LongestStreak        int       // Самая длинная серия
	LastActivity         time.Time // Последняя активность
	DaysActive           int       // Количество дней активности
	StrongestSkills      []string  // Самые сильные навыки
	WeakestSkills        []string  // Самые слабые навыки
	RecommendedExercises []string  // Рекомендованные упражнения
}

// NewProgressService создает новый сервис для работы с прогрессом
func NewProgressService(db *database.PostgresDB) *ProgressService {
	return &ProgressService{
		db: db,
	}
}

// GetUserStats получает статистику пользователя
func (s *ProgressService) GetUserStats(userID int64) (*UserStats, error) {
	// Получаем данные о прогрессе пользователя из БД
	progress, err := s.db.GetUserProgress(context.Background(), userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения прогресса пользователя: %w", err)
	}

	// Рассчитываем процент успешных упражнений
	successRate := 0.0
	if progress.TotalExercises > 0 {
		successRate = float64(progress.CorrectExercises) / float64(progress.TotalExercises) * 100
	}

	// Создаем статистику
	stats := &UserStats{
		TotalExercises:     progress.TotalExercises,
		CorrectExercises:   progress.CorrectExercises,
		SuccessRate:        successRate,
		TotalConversations: progress.TotalConversations,
		TotalMessages:      progress.TotalMessages,
		CurrentStreak:      progress.CurrentStreak,
		LongestStreak:      progress.LongestStreak,
		LastActivity:       progress.LastActivityDate,
	}

	// Рассчитываем количество дней активности
	// В реальном приложении нужно будет получить эту информацию из БД
	stats.DaysActive = progress.CurrentStreak

	// Анализируем сильные и слабые стороны пользователя
	// В реальном приложении нужно будет получить эту информацию из БД на основе упражнений
	stats.StrongestSkills = []string{"Vocabulary", "Reading"}
	stats.WeakestSkills = []string{"Grammar", "Listening"}

	// Рекомендуем упражнения на основе слабых сторон
	stats.RecommendedExercises = s.getRecommendedExercises(stats.WeakestSkills)

	return stats, nil
}

// getRecommendedExercises рекомендует упражнения на основе слабых сторон
func (s *ProgressService) getRecommendedExercises(weakestSkills []string) []string {
	recommendations := make([]string, 0, len(weakestSkills))

	for _, skill := range weakestSkills {
		switch skill {
		case "Grammar":
			recommendations = append(recommendations, "Practice verb tenses", "Work on conditional sentences")
		case "Vocabulary":
			recommendations = append(recommendations, "Learn new words daily", "Review vocabulary with flashcards")
		case "Listening":
			recommendations = append(recommendations, "Listen to English podcasts", "Watch English videos with subtitles")
		case "Speaking":
			recommendations = append(recommendations, "Practice conversations", "Record and listen to yourself speaking")
		case "Reading":
			recommendations = append(recommendations, "Read English articles", "Practice reading comprehension")
		case "Writing":
			recommendations = append(recommendations, "Write short essays", "Practice writing emails")
		}
	}

	return recommendations
}

// CalculateLevelProgress рассчитывает прогресс пользователя на текущем уровне
// Возвращает процент выполнения (0-100)
func (s *ProgressService) CalculateLevelProgress(userID int64, level string) (float64, error) {
	// В реальном приложении нужно будет получить данные из БД
	// Здесь используем упрощенный алгоритм

	// Получаем статистику пользователя
	stats, err := s.GetUserStats(userID)
	if err != nil {
		return 0, err
	}

	// Упрощенный алгоритм расчета прогресса
	// Зависит от количества упражнений и процента правильных ответов
	exercises := float64(stats.TotalExercises)
	successRate := stats.SuccessRate

	// Минимальное количество упражнений для перехода на следующий уровень
	var minExercises float64
	switch level {
	case "A1":
		minExercises = 50
	case "A2":
		minExercises = 100
	case "B1":
		minExercises = 150
	case "B2":
		minExercises = 200
	case "C1":
		minExercises = 250
	default:
		minExercises = 300
	}

	// Рассчитываем прогресс
	exerciseProgress := math.Min(exercises/minExercises, 1.0) * 70 // 70% от общего прогресса
	rateProgress := (successRate / 100) * 30                       // 30% от общего прогресса

	// Общий прогресс
	totalProgress := exerciseProgress + rateProgress

	// Округляем до двух знаков после запятой
	return math.Round(totalProgress*100) / 100, nil
}

// IsReadyForNextLevel проверяет, готов ли пользователь перейти на следующий уровень
func (s *ProgressService) IsReadyForNextLevel(userID int64, currentLevel string) (bool, string, error) {
	// Проверяем прогресс на текущем уровне
	progress, err := s.CalculateLevelProgress(userID, currentLevel)
	if err != nil {
		return false, "", err
	}

	// Если прогресс более 85%, пользователь готов перейти на следующий уровень
	if progress >= 85 {
		nextLevel := getNextLevel(currentLevel)
		return true, nextLevel, nil
	}

	return false, "", nil
}

// getNextLevel возвращает следующий уровень
func getNextLevel(currentLevel string) string {
	switch currentLevel {
	case "A1":
		return "A2"
	case "A2":
		return "B1"
	case "B1":
		return "B2"
	case "B2":
		return "C1"
	case "C1":
		return "C2"
	default:
		return "C2"
	}
}

// FormatProgressMessage форматирует сообщение о прогрессе пользователя
func (s *ProgressService) FormatProgressMessage(stats *UserStats, level string) string {
	// Рассчитываем прогресс на текущем уровне
	levelProgress, _ := s.CalculateLevelProgress(0, level) // userID 0 для примера

	// Форматируем сообщение
	message := fmt.Sprintf("📊 *Your Learning Progress*\n\n"+
		"*Current Level:* %s (%.0f%% completed)\n\n"+
		"*Statistics:*\n"+
		"• Exercises completed: %d\n"+
		"• Success rate: %.1f%%\n"+
		"• Conversations: %d\n"+
		"• Messages exchanged: %d\n"+
		"• Learning streak: %d days\n"+
		"• Longest streak: %d days\n\n"+
		"*Your Strengths:*\n",
		level, levelProgress,
		stats.TotalExercises,
		stats.SuccessRate,
		stats.TotalConversations,
		stats.TotalMessages,
		stats.CurrentStreak,
		stats.LongestStreak)

	// Добавляем сильные стороны
	for _, skill := range stats.StrongestSkills {
		message += fmt.Sprintf("✅ %s\n", skill)
	}

	message += "\n*Areas to Improve:*\n"

	// Добавляем слабые стороны
	for _, skill := range stats.WeakestSkills {
		message += fmt.Sprintf("⚠️ %s\n", skill)
	}

	message += "\n*Recommendations:*\n"

	// Добавляем рекомендации
	for _, rec := range stats.RecommendedExercises {
		message += fmt.Sprintf("• %s\n", rec)
	}

	// Проверяем, готов ли пользователь перейти на следующий уровень
	isReady, nextLevel, _ := s.IsReadyForNextLevel(0, level) // userID 0 для примера
	if isReady {
		message += fmt.Sprintf("\n🎉 *Congratulations!* You're ready to advance to level %s!\n", nextLevel)
	}

	return message
}
