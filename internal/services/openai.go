package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// OpenAIService предоставляет функциональность для работы с OpenAI API
type OpenAIService struct {
	apiKey string
	client *http.Client
}

// OpenAIRequest представляет запрос к API ChatGPT
type OpenAIRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
}

// ChatMessage представляет сообщение в диалоге ChatGPT
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIResponse представляет ответ от ChatGPT API
type OpenAIResponse struct {
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
	Error *OpenAIError `json:"error,omitempty"`
}

// OpenAIError представляет структуру ошибки OpenAI API
type OpenAIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

// NewOpenAIService создает новый сервис для работы с OpenAI
func NewOpenAIService(apiKey string) *OpenAIService {
	return &OpenAIService{
		apiKey: apiKey,
		client: &http.Client{},
	}
}

// GenerateResponse отправляет запрос к API ChatGPT и получает ответ
func (s *OpenAIService) GenerateResponse(prompt string, systemPrompt string) (string, error) {
	messages := []ChatMessage{
		{
			Role:    "system",
			Content: systemPrompt,
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	return s.SendChatRequest(messages)
}

// SendChatRequest отправляет запрос к ChatGPT API
func (s *OpenAIService) SendChatRequest(messages []ChatMessage) (string, error) {
	reqBody := OpenAIRequest{
		Model:    "gpt-3.5-turbo", // Можно изменить на другую модель
		Messages: messages,
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("ошибка маршалинга JSON: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(reqJSON))
	if err != nil {
		return "", fmt.Errorf("ошибка создания запроса: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ошибка отправки запроса: %w", err)
	}
	defer resp.Body.Close()

	var response OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("ошибка декодирования ответа: %w", err)
	}

	if response.Error != nil {
		return "", fmt.Errorf("ошибка API: %s (%s)", response.Error.Message, response.Error.Type)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("пустой ответ от API")
	}

	return response.Choices[0].Message.Content, nil
}

// CheckGrammar проверяет грамматику текста с помощью ChatGPT
func (s *OpenAIService) CheckGrammar(text string) (string, error) {
	systemPrompt := `You are a helpful English language assistant. Your task is to:
1. Identify grammar, spelling, and style errors in the provided text
2. Provide corrections with explanations
3. Rate the overall proficiency level (A1, A2, B1, B2, C1, C2)
Format your response in clear sections.`

	return s.GenerateResponse(text, systemPrompt)
}

// GenerateExercise создает упражнение заданного уровня сложности
func (s *OpenAIService) GenerateExercise(exerciseType string, level string) (string, error) {
	systemPrompt := fmt.Sprintf(`You are an English language tutor. Create a %s exercise for %s level student. 
The exercise should be challenging but appropriate for the level.
Format your response clearly with instructions and examples if needed.`, exerciseType, level)

	return s.GenerateResponse("Generate an exercise", systemPrompt)
}

// SimulateConversation поддерживает диалог на заданную тему
func (s *OpenAIService) SimulateConversation(userMessage string, conversationHistory []ChatMessage) (string, error) {
	// Добавляем системный промпт для разговора
	if len(conversationHistory) == 0 {
		conversationHistory = append(conversationHistory, ChatMessage{
			Role:    "system",
			Content: "You are a helpful English tutor having a conversation with a student learning English. Keep your responses friendly, encouraging, and adapted to their level. Use simple language for beginners and more complex structures for advanced students.",
		})
	}

	// Добавляем сообщение пользователя
	conversationHistory = append(conversationHistory, ChatMessage{
		Role:    "user",
		Content: userMessage,
	})

	// Отправляем запрос с полной историей диалога
	return s.SendChatRequest(conversationHistory)
}
