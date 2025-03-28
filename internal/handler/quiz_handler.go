package handler

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/trivia-api/internal/domain/entity"
	"github.com/yourusername/trivia-api/internal/service"
	"github.com/yourusername/trivia-api/internal/service/quizmanager"
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
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid quiz ID"})
		return
	}

	quiz, err := h.quizService.GetQuizByID(uint(id))
	if err != nil {
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

	log.Printf("[QuizHandler] Запланированные викторины перед отправкой JSON (количество: %d): %+v", len(quizzes), quizzes)
	c.JSON(http.StatusOK, quizzes)
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
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid quiz ID"})
		return
	}

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

	if err := h.quizService.AddQuestions(uint(id), questions); err != nil {
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
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid quiz ID"})
		return
	}

	var req ScheduleQuizRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Сначала обновляем время в базе данных
	if err := h.quizService.ScheduleQuiz(uint(id), req.ScheduledTime); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Затем планируем викторину через QuizManager
	if err := h.quizManager.ScheduleQuiz(uint(id), req.ScheduledTime); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Quiz scheduled successfully"})
}

// CancelQuiz обрабатывает запрос на отмену викторины
func (h *QuizHandler) CancelQuiz(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid quiz ID"})
		return
	}

	if err := h.quizManager.CancelQuiz(uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Quiz cancelled successfully"})
}

// GetQuizWithQuestions возвращает викторину вместе с вопросами
func (h *QuizHandler) GetQuizWithQuestions(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid quiz ID"})
		return
	}

	quiz, err := h.quizService.GetQuizWithQuestions(uint(id))
	if err != nil {
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

	// Преобразуем формат вопросов для фронтенда
	type QuestionResponse struct {
		ID            uint                         `json:"id"`
		QuizID        uint                         `json:"quiz_id"`
		Text          string                       `json:"text"`
		Options       []quizmanager.QuestionOption `json:"options"`
		CorrectOption int                          `json:"-"`
		TimeLimitSec  int                          `json:"time_limit_sec"`
		PointValue    int                          `json:"point_value"`
		CreatedAt     time.Time                    `json:"created_at"`
		UpdatedAt     time.Time                    `json:"updated_at"`
	}

	type QuizResponse struct {
		ID            uint               `json:"id"`
		Title         string             `json:"title"`
		Description   string             `json:"description"`
		ScheduledTime time.Time          `json:"scheduled_time"`
		Status        string             `json:"status"`
		QuestionCount int                `json:"question_count"`
		Questions     []QuestionResponse `json:"questions,omitempty"`
		CreatedAt     time.Time          `json:"created_at"`
		UpdatedAt     time.Time          `json:"updated_at"`
	}

	// Преобразуем в формат для ответа
	response := QuizResponse{
		ID:            quiz.ID,
		Title:         quiz.Title,
		Description:   quiz.Description,
		ScheduledTime: quiz.ScheduledTime,
		Status:        quiz.Status,
		QuestionCount: quiz.QuestionCount,
		CreatedAt:     quiz.CreatedAt,
		UpdatedAt:     quiz.UpdatedAt,
		Questions:     make([]QuestionResponse, len(quiz.Questions)),
	}

	// Преобразуем options для каждого вопроса
	for i, q := range quiz.Questions {
		response.Questions[i] = QuestionResponse{
			ID:            q.ID,
			QuizID:        q.QuizID,
			Text:          q.Text,
			Options:       quizmanager.ConvertOptionsToObjects(q.Options),
			CorrectOption: q.CorrectOption,
			TimeLimitSec:  q.TimeLimitSec,
			PointValue:    q.PointValue,
			CreatedAt:     q.CreatedAt,
			UpdatedAt:     q.UpdatedAt,
		}
	}

	c.JSON(http.StatusOK, response)
}

// GetQuizResults возвращает результаты викторины
func (h *QuizHandler) GetQuizResults(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid quiz ID"})
		return
	}

	results, err := h.resultService.GetQuizResults(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, results)
}

// GetUserQuizResult возвращает результат пользователя для конкретной викторины
func (h *QuizHandler) GetUserQuizResult(c *gin.Context) {
	quizIDStr := c.Param("id")
	quizID, err := strconv.ParseUint(quizIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid quiz ID"})
		return
	}

	// Получаем ID пользователя из контекста
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	result, err := h.resultService.GetUserResult(userID.(uint), uint(quizID))
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
