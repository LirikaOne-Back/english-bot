package services

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// ExerciseService предоставляет функциональность для работы с упражнениями
type ExerciseService struct {
	openAI *OpenAIService
}

// ExerciseType определяет тип упражнения
type ExerciseType string

const (
	ExerciseTypeGrammar     ExerciseType = "grammar"     // Грамматические упражнения
	ExerciseTypeVocabulary  ExerciseType = "vocabulary"  // Словарные упражнения
	ExerciseTypeTranslation ExerciseType = "translation" // Упражнения на перевод
	ExerciseTypeListening   ExerciseType = "listening"   // Упражнения на аудирование
	ExerciseTypeSpeaking    ExerciseType = "speaking"    // Упражнения на разговорную речь
)

// EnglishLevel определяет уровень владения английским
type EnglishLevel string

const (
	EnglishLevelA1 EnglishLevel = "A1" // Beginner
	EnglishLevelA2 EnglishLevel = "A2" // Elementary
	EnglishLevelB1 EnglishLevel = "B1" // Intermediate
	EnglishLevelB2 EnglishLevel = "B2" // Upper Intermediate
	EnglishLevelC1 EnglishLevel = "C1" // Advanced
	EnglishLevelC2 EnglishLevel = "C2" // Proficiency
)

// Exercise представляет упражнение
type Exercise struct {
	Type        ExerciseType // Тип упражнения
	Level       EnglishLevel // Уровень сложности
	Instruction string       // Инструкция к упражнению
	Content     string       // Содержание упражнения
	Answer      string       // Правильный ответ
	Options     []string     // Варианты ответов (для выбора)
}

// NewExerciseService создает новый сервис для работы с упражнениями
func NewExerciseService(openAI *OpenAIService) *ExerciseService {
	return &ExerciseService{
		openAI: openAI,
	}
}

// GetPromptForExerciseType возвращает системный промпт для генерации упражнения
func (s *ExerciseService) GetPromptForExerciseType(exerciseType ExerciseType, level EnglishLevel) string {
	switch exerciseType {
	case ExerciseTypeGrammar:
		return fmt.Sprintf(`Create a grammar exercise for %s level student. 
The exercise should test a specific grammar point appropriate for this level.
The response should include:
1. Clear instructions
2. The exercise content
3. The correct answer(s)
4. A brief explanation of the grammar rule tested
Format the response clearly with sections.`, level)

	case ExerciseTypeVocabulary:
		return fmt.Sprintf(`Create a vocabulary exercise for %s level student.
The exercise should test knowledge of words appropriate for this level.
The response should include:
1. Clear instructions
2. The exercise content (could be fill-in-the-blank, matching, etc.)
3. The correct answer(s)
4. Usage examples for the vocabulary items
Format the response clearly with sections.`, level)

	case ExerciseTypeTranslation:
		return fmt.Sprintf(`Create a translation exercise for %s level student.
Provide 3-5 sentences in Russian that the student should translate to English.
The sentences should be appropriate for this level and test specific grammar/vocabulary.
The response should include:
1. Clear instructions
2. The sentences to translate (in Russian)
3. The correct English translations
4. Notes on any particularly challenging aspects
Format the response clearly with sections.`, level)

	default:
		return fmt.Sprintf(`Create an English language exercise for %s level student.
The exercise should be appropriate for this level and engaging.
The response should include:
1. Clear instructions
2. The exercise content
3. The correct answer(s) or evaluation criteria
Format the response clearly with sections.`, level)
	}
}

// GenerateExercise генерирует упражнение через OpenAI
func (s *ExerciseService) GenerateExercise(exerciseType ExerciseType, level EnglishLevel) (*Exercise, error) {
	// Получаем промпт для генерации упражнения
	prompt := s.GetPromptForExerciseType(exerciseType, level)

	// Генерируем упражнение через OpenAI
	content, err := s.openAI.GenerateResponse("Generate an exercise", prompt)
	if err != nil {
		return nil, fmt.Errorf("ошибка генерации упражнения: %w", err)
	}

	// Создаем упражнение
	exercise := &Exercise{
		Type:        exerciseType,
		Level:       level,
		Content:     content,
		Instruction: extractInstructions(content),
		// Здесь мы должны извлечь ответ из сгенерированного контента,
		// но это требует более сложного парсинга
	}

	return exercise, nil
}

// GenerateSimpleExercise генерирует простое упражнение без использования OpenAI
// Полезно как запасной вариант или для тестирования
func (s *ExerciseService) GenerateSimpleExercise(exerciseType ExerciseType, level EnglishLevel) (*Exercise, error) {
	// Инициализируем генератор случайных чисел
	rand.Seed(time.Now().UnixNano())

	var exercise Exercise
	exercise.Type = exerciseType
	exercise.Level = level

	switch exerciseType {
	case ExerciseTypeGrammar:
		// Генерируем упражнение на времена для уровней A1-B1
		if level == EnglishLevelA1 || level == EnglishLevelA2 || level == EnglishLevelB1 {
			exercise.Instruction = "Choose the correct form of the verb to complete the sentence."

			// Простые предложения Present Simple vs Present Continuous
			sentences := []string{
				"I usually (go/am going) to work by bus.",
				"She (speaks/is speaking) on the phone right now.",
				"They (don't like/aren't liking) coffee.",
				"What (do you do/are you doing) this weekend?",
				"He (doesn't work/isn't working) today because he is sick.",
			}

			answers := []string{
				"go",
				"is speaking",
				"don't like",
				"are you doing",
				"isn't working",
			}

			// Выбираем случайное предложение
			index := rand.Intn(len(sentences))
			exercise.Content = sentences[index]
			exercise.Answer = answers[index]

			// Создаем варианты ответов (извлекаем из скобок)
			options := strings.Split(extractOptions(exercise.Content), "/")
			exercise.Options = options

			// Очищаем контент от скобок с вариантами
			exercise.Content = cleanExerciseContent(exercise.Content)
		} else {
			// Для более высоких уровней - сложные условные предложения
			exercise.Instruction = "Complete the conditional sentence with the correct form of the verb in brackets."

			sentences := []string{
				"If I (have) more time, I would learn another language.",
				"She would have passed the exam if she (study) harder.",
				"If you (call) me earlier, I would have picked you up.",
				"What would you do if you (win) the lottery?",
				"He (travel) around the world if he didn't have to work.",
			}

			answers := []string{
				"had",
				"had studied",
				"had called",
				"won",
				"would travel",
			}

			// Выбираем случайное предложение
			index := rand.Intn(len(sentences))
			exercise.Content = sentences[index]
			exercise.Answer = answers[index]
		}

	case ExerciseTypeVocabulary:
		// Упражнение на словарный запас
		exercise.Instruction = "Fill in the blank with the correct word from the options."

		if level == EnglishLevelA1 || level == EnglishLevelA2 {
			// Простые слова для начинающих
			sentences := []string{
				"I need to _____ (buy/sell/give) some food for dinner.",
				"She _____ (lives/works/studies) in London with her family.",
				"We usually _____ (have/take/do) breakfast at 8 AM.",
				"They don't _____ (like/want/need) to watch TV in the evening.",
				"Can you _____ (open/close/lock) the window, please?",
			}

			answers := []string{
				"buy",
				"lives",
				"have",
				"like",
				"open",
			}

			index := rand.Intn(len(sentences))
			exercise.Content = sentences[index]
			exercise.Answer = answers[index]
			exercise.Options = strings.Split(extractOptions(exercise.Content), "/")
			exercise.Content = cleanExerciseContent(exercise.Content)
		} else {
			// Более сложные слова для продвинутых
			sentences := []string{
				"The government implemented _____ (stringent/lenient/ambiguous) measures to control the spread of the virus.",
				"Her _____ (eloquent/reticent/verbose) speech captivated the entire audience.",
				"The scandal had a _____ (detrimental/beneficial/neutral) effect on his reputation.",
				"Scientists have _____ (corroborated/refuted/ignored) the theory with new evidence.",
				"The company is facing _____ (unprecedented/expected/minimal) challenges due to economic changes.",
			}

			answers := []string{
				"stringent",
				"eloquent",
				"detrimental",
				"corroborated",
				"unprecedented",
			}

			index := rand.Intn(len(sentences))
			exercise.Content = sentences[index]
			exercise.Answer = answers[index]
			exercise.Options = strings.Split(extractOptions(exercise.Content), "/")
			exercise.Content = cleanExerciseContent(exercise.Content)
		}

	case ExerciseTypeTranslation:
		// Упражнение на перевод
		exercise.Instruction = "Translate the following sentence into English."

		if level == EnglishLevelA1 || level == EnglishLevelA2 {
			sentences := []string{
				"Меня зовут Иван. Я живу в Москве.",
				"У меня есть собака и кошка.",
				"Я люблю пиццу и мороженое.",
				"Сегодня хорошая погода.",
				"Я учу английский язык два года.",
			}

			answers := []string{
				"My name is Ivan. I live in Moscow.",
				"I have a dog and a cat.",
				"I like/love pizza and ice cream.",
				"The weather is good today.",
				"I have been learning English for two years.",
			}

			index := rand.Intn(len(sentences))
			exercise.Content = sentences[index]
			exercise.Answer = answers[index]
		} else {
			sentences := []string{
				"Несмотря на все трудности, он продолжал идти к своей цели.",
				"Если бы я знал об этом раньше, я бы принял другое решение.",
				"Чем больше я об этом думаю, тем меньше мне это нравится.",
				"Компания объявила о сокращении штата из-за экономического кризиса.",
				"Необходимо разработать комплексный подход к решению данной проблемы.",
			}

			answers := []string{
				"Despite all the difficulties, he continued moving towards his goal.",
				"If I had known about this earlier, I would have made a different decision.",
				"The more I think about it, the less I like it.",
				"The company announced staff reductions due to the economic crisis.",
				"It is necessary to develop a comprehensive approach to solving this problem.",
			}

			index := rand.Intn(len(sentences))
			exercise.Content = sentences[index]
			exercise.Answer = answers[index]
		}
	}

	return &exercise, nil
}

// CheckAnswer проверяет ответ пользователя
// Возвращает оценку (0-100) и комментарий
func (s *ExerciseService) CheckAnswer(exercise *Exercise, userAnswer string) (int, string) {
	// Очищаем ответы от лишних пробелов, приводим к нижнему регистру
	normalizedUserAnswer := strings.ToLower(strings.TrimSpace(userAnswer))
	normalizedCorrectAnswer := strings.ToLower(strings.TrimSpace(exercise.Answer))

	// Подготавливаем возможные варианты правильных ответов
	// Некоторые ответы могут иметь несколько вариантов (например, "like/love")
	correctVariants := strings.Split(normalizedCorrectAnswer, "/")

	// Проверяем точное совпадение с одним из вариантов
	for _, variant := range correctVariants {
		if normalizedUserAnswer == strings.TrimSpace(variant) {
			return 100, "Perfect! Your answer is correct."
		}
	}

	// Проверяем на опечатки, ошибки в словах
	for _, variant := range correctVariants {
		// Если совпадение более 80% (простая эвристика)
		if levenshteinRatio(normalizedUserAnswer, strings.TrimSpace(variant)) > 0.8 {
			return 80, "Almost correct! There are some minor errors in your answer."
		}
	}

	// Проверяем на частичное совпадение
	for _, variant := range correctVariants {
		if strings.Contains(normalizedUserAnswer, strings.TrimSpace(variant)) {
			return 60, "Partially correct. Your answer contains the right elements but has some issues."
		}
	}

	return 0, "Your answer is incorrect. Please try again."
}

// Вспомогательные функции

// extractInstructions извлекает инструкции из сгенерированного контента
func extractInstructions(content string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), "instruct") || strings.Contains(line, "1.") {
			if i+1 < len(lines) {
				return strings.TrimSpace(lines[i+1])
			}
			return strings.TrimSpace(line)
		}
	}

	// Если ничего не нашли, возвращаем первую строку
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}

	return ""
}

// extractOptions извлекает варианты ответов из скобок
func extractOptions(content string) string {
	start := strings.Index(content, "(")
	end := strings.Index(content, ")")

	if start != -1 && end != -1 && start < end {
		return content[start+1 : end]
	}

	return ""
}

// cleanExerciseContent очищает контент от скобок с вариантами
func cleanExerciseContent(content string) string {
	start := strings.Index(content, "(")
	end := strings.Index(content, ")")

	if start != -1 && end != -1 && start < end {
		return content[:start] + "_____ " + content[end+1:]
	}

	return content
}

// levenshteinRatio вычисляет коэффициент сходства строк на основе расстояния Левенштейна
// Возвращает значение от 0 до 1, где 1 означает полное совпадение
func levenshteinRatio(s1, s2 string) float64 {
	dist := levenshteinDistance(s1, s2)
	maxLen := float64(max(len(s1), len(s2)))

	if maxLen == 0 {
		return 1.0
	}

	return 1.0 - float64(dist)/maxLen
}

// levenshteinDistance вычисляет расстояние Левенштейна между двумя строками
func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Создаем матрицу расстояний
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
		matrix[i][0] = i
	}
	for j := range matrix[0] {
		matrix[0][j] = j
	}

	// Заполняем матрицу
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}
			matrix[i][j] = min(
				matrix[i-1][j]+1,      // удаление
				matrix[i][j-1]+1,      // вставка
				matrix[i-1][j-1]+cost, // замена
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

// min возвращает минимальное из трех чисел
func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// max возвращает максимальное из двух чисел
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
