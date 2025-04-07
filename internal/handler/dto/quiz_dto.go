package dto

import (
	"time"

	"github.com/yourusername/trivia-api/internal/domain/entity" // Используем правильный путь модуля
	"github.com/yourusername/trivia-api/internal/handler/helper"
)

// QuestionResponse представляет вопрос в формате для ответа клиенту
type QuestionResponse struct {
	ID           uint                    `json:"id"`
	QuizID       uint                    `json:"quiz_id"`
	Text         string                  `json:"text"`
	Options      []helper.QuestionOption `json:"options"`
	TimeLimitSec int                     `json:"time_limit_sec"`
	PointValue   int                     `json:"point_value"`
	CreatedAt    time.Time               `json:"created_at"`
	UpdatedAt    time.Time               `json:"updated_at"`
}

// QuizResponse представляет викторину в формате для ответа клиенту
type QuizResponse struct {
	ID            uint               `json:"id"`
	Title         string             `json:"title"`
	Description   string             `json:"description,omitempty"`
	ScheduledTime time.Time          `json:"scheduled_time"`
	Status        string             `json:"status"`
	Questions     []QuestionResponse `json:"questions,omitempty"` // Слайс DTO вопросов
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
}

// NewQuestionResponse создает DTO для вопроса
// Примечание: Эта функция используется внутри NewQuizResponse
func NewQuestionResponse(q *entity.Question) QuestionResponse {
	// Используем хелпер для преобразования опций
	optionsDTO := helper.ConvertOptionsToObjects(q.Options)

	// Логика скрытия CorrectOption остается в вызывающем коде (хэндлере).
	return QuestionResponse{
		ID:           q.ID,
		QuizID:       q.QuizID,
		Text:         q.Text,
		Options:      optionsDTO, // Используем результат хелпера
		TimeLimitSec: q.TimeLimitSec,
		PointValue:   q.PointValue,
		CreatedAt:    q.CreatedAt,
		UpdatedAt:    q.UpdatedAt,
	}
}

// NewQuizResponse создает DTO для викторины
func NewQuizResponse(quiz *entity.Quiz) *QuizResponse {
	if quiz == nil {
		return nil
	}

	questionsDTO := make([]QuestionResponse, len(quiz.Questions))
	for i, q := range quiz.Questions {
		// Важно: передаем указатель на элемент слайса
		questionsDTO[i] = NewQuestionResponse(&q)
	}

	return &QuizResponse{
		ID:            quiz.ID,
		Title:         quiz.Title,
		Description:   quiz.Description,
		ScheduledTime: quiz.ScheduledTime,
		Status:        string(quiz.Status), // Преобразуем статус в строку
		Questions:     questionsDTO,
		CreatedAt:     quiz.CreatedAt,
		UpdatedAt:     quiz.UpdatedAt,
	}
}
