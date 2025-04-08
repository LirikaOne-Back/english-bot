package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// LanguageToolService предоставляет функциональность для работы с LanguageTool API
type LanguageToolService struct {
	baseURL string
	client  *http.Client
}

// LanguageToolRequest представляет запрос к API LanguageTool
type LanguageToolRequest struct {
	Text          string `json:"text"`
	Language      string `json:"language"`
	EnabledOnly   bool   `json:"enabledOnly,omitempty"`
	Level         string `json:"level,omitempty"`
	MotherTongue  string `json:"motherTongue,omitempty"`
	PremiumOnly   bool   `json:"premiumOnly,omitempty"`
	DisabledRules string `json:"disabledRules,omitempty"`
}

// LanguageToolMatch представляет найденную ошибку
type LanguageToolMatch struct {
	Message      string `json:"message"`
	Shortmessage string `json:"shortMessage"`
	Offset       int    `json:"offset"`
	Length       int    `json:"length"`
	Rule         struct {
		ID          string `json:"id"`
		Description string `json:"description"`
		IssueType   string `json:"issueType"`
		Category    struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"category"`
	} `json:"rule"`
	Replacements []struct {
		Value string `json:"value"`
	} `json:"replacements"`
	Context struct {
		Text   string `json:"text"`
		Offset int    `json:"offset"`
		Length int    `json:"length"`
	} `json:"context"`
	Sentence string `json:"sentence"`
}

// LanguageToolResponse представляет ответ от LanguageTool API
type LanguageToolResponse struct {
	Software struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"software"`
	Matches  []LanguageToolMatch `json:"matches"`
	Language struct {
		Name         string `json:"name"`
		Code         string `json:"code"`
		DetectedCode string `json:"detectedLanguage,omitempty"`
	} `json:"language"`
}

// NewLanguageToolService создает новый сервис для работы с LanguageTool
func NewLanguageToolService() *LanguageToolService {
	return &LanguageToolService{
		baseURL: "https://api.languagetool.org/v2/check",
		client:  &http.Client{},
	}
}

// CheckText проверяет текст на грамматические и стилистические ошибки
func (s *LanguageToolService) CheckText(text string) (*LanguageToolResponse, error) {
	// Формируем данные для запроса
	data := url.Values{}
	data.Set("text", text)
	data.Set("language", "en-US")
	data.Set("enabledOnly", "false")

	// Отправляем запрос
	req, err := http.NewRequest("POST", s.baseURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка отправки запроса: %w", err)
	}
	defer resp.Body.Close()

	// Проверяем код ответа
	if resp.StatusCode != http.StatusOK {
		var errorBody bytes.Buffer
		errorBody.ReadFrom(resp.Body)
		return nil, fmt.Errorf("ошибка API (%d): %s", resp.StatusCode, errorBody.String())
	}

	// Разбираем ответ
	var response LanguageToolResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("ошибка декодирования ответа: %w", err)
	}

	return &response, nil
}

// FormatCorrections форматирует найденные ошибки в удобный для пользователя вид
func (s *LanguageToolService) FormatCorrections(text string, response *LanguageToolResponse) string {
	if len(response.Matches) == 0 {
		return "✅ Ваш текст грамматически корректен! Отличная работа!"
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("🔍 Найдено %d ошибок:\n\n", len(response.Matches)))

	for i, match := range response.Matches {
		// Добавляем номер ошибки
		result.WriteString(fmt.Sprintf("%d. *Ошибка*: %s\n", i+1, match.Message))

		// Добавляем контекст с выделенной ошибкой
		errorText := text[match.Offset : match.Offset+match.Length]
		contextBefore := ""
		contextAfter := ""

		if match.Offset > 10 {
			contextBefore = "..." + text[match.Offset-10:match.Offset]
		} else {
			contextBefore = text[:match.Offset]
		}

		if match.Offset+match.Length+10 < len(text) {
			contextAfter = text[match.Offset+match.Length:match.Offset+match.Length+10] + "..."
		} else {
			contextAfter = text[match.Offset+match.Length:]
		}

		result.WriteString(fmt.Sprintf("   *Контекст*: %s*%s*%s\n", contextBefore, errorText, contextAfter))

		// Добавляем предлагаемые исправления
		if len(match.Replacements) > 0 {
			replacements := make([]string, 0, len(match.Replacements))
			for j, r := range match.Replacements {
				if j < 3 { // Ограничиваем количество предложений
					replacements = append(replacements, r.Value)
				}
			}
			result.WriteString(fmt.Sprintf("   *Варианты исправления*: %s\n", strings.Join(replacements, ", ")))
		}

		// Добавляем пустую строку между ошибками
		result.WriteString("\n")
	}

	return result.String()
}

// CheckGrammar комбинирует проверку и форматирование результатов
func (s *LanguageToolService) CheckGrammar(text string) (string, error) {
	response, err := s.CheckText(text)
	if err != nil {
		return "", err
	}

	return s.FormatCorrections(text, response), nil
}
