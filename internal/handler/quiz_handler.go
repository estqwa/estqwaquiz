package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/trivia-api/internal/domain/entity"
	"github.com/yourusername/trivia-api/internal/handler/dto"
	"github.com/yourusername/trivia-api/internal/service"
)

// QuizHandler обрабатывает запросы, связанные с викторинами
type QuizHandler struct {
	quizService   *service.QuizService
	resultService *service.ResultService
	quizManager   *service.QuizManager
}

// NewQuizHandler создает новый обработчик викторин
func NewQuizHandler(
	quizService *service.QuizService,
	resultService *service.ResultService,
	quizManager *service.QuizManager,
) *QuizHandler {
	return &QuizHandler{
		quizService:   quizService,
		resultService: resultService,
		quizManager:   quizManager,
	}
}

// CreateQuizRequest представляет запрос на создание викторины
type CreateQuizRequest struct {
	Title         string    `json:"title" binding:"required,min=3,max=100"`
	Description   string    `json:"description" binding:"omitempty,max=500"`
	ScheduledTime time.Time `json:"scheduled_time" binding:"required"`
}

// CreateQuiz обрабатывает запрос на создание викторины
func (h *QuizHandler) CreateQuiz(c *gin.Context) {
	var req CreateQuizRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	quiz, err := h.quizService.CreateQuiz(req.Title, req.Description, req.ScheduledTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, quiz)
}

// GetQuiz возвращает информацию о викторине
func (h *QuizHandler) GetQuiz(c *gin.Context) {
	quizID := c.MustGet("quizID").(uint) // Получаем из контекста

	quiz, err := h.quizService.GetQuizByID(quizID)
	if err != nil {
		// TODO: Улучшить обработку ошибок (п.7)
		c.JSON(http.StatusNotFound, gin.H{"error": "Quiz not found"})
		return
	}

	c.JSON(http.StatusOK, quiz)
}

// GetActiveQuiz возвращает информацию об активной викторине
func (h *QuizHandler) GetActiveQuiz(c *gin.Context) {
	// Проверяем сначала в QuizManager
	activeQuiz := h.quizManager.GetActiveQuiz()
	if activeQuiz != nil {
		c.JSON(http.StatusOK, activeQuiz)
		return
	}

	// Если не найдена активная викторина у менеджера, ищем в БД
	quiz, err := h.quizService.GetActiveQuiz()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No active quiz"})
		return
	}

	c.JSON(http.StatusOK, quiz)
}

// GetScheduledQuizzes возвращает список запланированных викторин
func (h *QuizHandler) GetScheduledQuizzes(c *gin.Context) {
	quizzes, err := h.quizService.GetScheduledQuizzes()
	if err != nil {
		log.Printf("[QuizHandler] Ошибка при получении запланированных викторин: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("[QuizHandler] Запланированные викторины перед маршалингом (количество: %d): %+v", len(quizzes), quizzes)

	// Явный маршалинг в JSON
	jsonData, err := json.Marshal(quizzes)
	if err != nil {
		log.Printf("[QuizHandler] Ошибка при маршалинге JSON: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error during JSON marshaling"})
		return
	}

	// Логирование результата маршалинга
	log.Printf("[QuizHandler] Сериализованные JSON данные: %s", string(jsonData))

	// Отправка сырых JSON байт
	c.Data(http.StatusOK, "application/json; charset=utf-8", jsonData)
}

// AddQuestionsRequest представляет запрос на добавление вопросов
type AddQuestionsRequest struct {
	Questions []struct {
		Text          string   `json:"text" binding:"required,min=3,max=500"`
		Options       []string `json:"options" binding:"required,min=2,max=5"`
		CorrectOption int      `json:"correct_option" binding:"required,min=0"`
		TimeLimitSec  int      `json:"time_limit_sec" binding:"required,min=5,max=60"`
		PointValue    int      `json:"point_value" binding:"required,min=1,max=100"`
	} `json:"questions" binding:"required,min=1"`
}

// AddQuestions обрабатывает запрос на добавление вопросов к викторине
func (h *QuizHandler) AddQuestions(c *gin.Context) {
	quizID := c.MustGet("quizID").(uint) // Получаем из контекста

	var req AddQuestionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Преобразуем данные в формат для сервиса
	questions := make([]entity.Question, 0, len(req.Questions))
	for _, q := range req.Questions {
		questions = append(questions, entity.Question{
			Text:          q.Text,
			Options:       entity.StringArray(q.Options),
			CorrectOption: q.CorrectOption,
			TimeLimitSec:  q.TimeLimitSec,
			PointValue:    q.PointValue,
		})
	}

	if err := h.quizService.AddQuestions(quizID, questions); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Questions added successfully"})
}

// ScheduleQuizRequest представляет запрос на планирование викторины
type ScheduleQuizRequest struct {
	ScheduledTime time.Time `json:"scheduled_time" binding:"required"`
}

// ScheduleQuiz обрабатывает запрос на планирование времени викторины
func (h *QuizHandler) ScheduleQuiz(c *gin.Context) {
	quizID := c.MustGet("quizID").(uint) // Получаем из контекста

	var req ScheduleQuizRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Сначала обновляем время в базе данных
	if err := h.quizService.ScheduleQuiz(quizID, req.ScheduledTime); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Затем планируем викторину через QuizManager
	if err := h.quizManager.ScheduleQuiz(quizID, req.ScheduledTime); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Quiz scheduled successfully"})
}

// CancelQuiz обрабатывает запрос на отмену викторины
func (h *QuizHandler) CancelQuiz(c *gin.Context) {
	quizID := c.MustGet("quizID").(uint) // Получаем из контекста

	if err := h.quizManager.CancelQuiz(quizID); err != nil {
		// TODO: Улучшить обработку ошибок (п.7)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Quiz cancelled successfully"})
}

// GetQuizWithQuestions возвращает викторину вместе с вопросами
func (h *QuizHandler) GetQuizWithQuestions(c *gin.Context) {
	quizID := c.MustGet("quizID").(uint) // Получаем из контекста

	quiz, err := h.quizService.GetQuizWithQuestions(quizID)
	if err != nil {
		// TODO: Улучшить обработку ошибок (п.7)
		c.JSON(http.StatusNotFound, gin.H{"error": "Quiz not found"})
		return
	}

	// Скрываем правильные ответы от клиентов,
	// если викторина еще не завершена
	if !quiz.IsCompleted() {
		for i := range quiz.Questions {
			quiz.Questions[i].CorrectOption = -1
		}
	}

	// Используем конструктор DTO
	response := dto.NewQuizResponse(quiz)

	c.JSON(http.StatusOK, response)
}

// GetQuizResults возвращает результаты викторины
func (h *QuizHandler) GetQuizResults(c *gin.Context) {
	quizID := c.MustGet("quizID").(uint) // Получаем из контекста

	results, err := h.resultService.GetQuizResults(quizID)
	if err != nil {
		// TODO: Улучшить обработку ошибок (п.7)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, results)
}

// GetUserQuizResult возвращает результат пользователя для конкретной викторины
func (h *QuizHandler) GetUserQuizResult(c *gin.Context) {
	quizID := c.MustGet("quizID").(uint) // Получаем из контекста

	// Получаем ID пользователя из контекста
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	result, err := h.resultService.GetUserResult(userID.(uint), quizID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Result not found"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ListQuizzes возвращает список викторин с пагинацией
func (h *QuizHandler) ListQuizzes(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	quizzes, err := h.quizService.ListQuizzes(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, quizzes)
}

// handleQuizError обрабатывает ошибки от сервисов викторин и отправляет соответствующий HTTP ответ
func (h *QuizHandler) handleQuizError(c *gin.Context, err error) {
	// Определяем тип ошибки и возвращаем соответствующий статус
	// TODO: Определить и использовать специфичные типы ошибок из сервисов
	if errors.Is(err, service.ErrQuizNotFound) { // Пример кастомной ошибки
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	} else if errors.Is(err, service.ErrQuizNotSchedulable) { // Пример кастомной ошибки
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	} else if errors.Is(err, service.ErrValidation) { // Пример кастомной ошибки
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	} else {
		// Общая ошибка сервера
		log.Printf("ERROR: Internal server error in QuizHandler: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
	}
}
