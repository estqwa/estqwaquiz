package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/yourusername/trivia-api/internal/domain/entity"
	"github.com/yourusername/trivia-api/internal/domain/repository"
)

// Максимальное количество вопросов в викторине
const MaxQuizQuestions = 10

// QuizService предоставляет методы для работы с викторинами
type QuizService struct {
	quizRepo     repository.QuizRepository
	questionRepo repository.QuestionRepository
	cacheRepo    repository.CacheRepository
}

// NewQuizService создает новый сервис викторин
func NewQuizService(
	quizRepo repository.QuizRepository,
	questionRepo repository.QuestionRepository,
	cacheRepo repository.CacheRepository,
) *QuizService {
	return &QuizService{
		quizRepo:     quizRepo,
		questionRepo: questionRepo,
		cacheRepo:    cacheRepo,
	}
}

// CreateQuiz создает новую викторину
func (s *QuizService) CreateQuiz(title, description string, scheduledTime time.Time) (*entity.Quiz, error) {
	// Проверяем, что время проведения в будущем
	if scheduledTime.Before(time.Now()) {
		return nil, errors.New("scheduled time must be in the future")
	}

	// Создаем новую викторину
	quiz := &entity.Quiz{
		Title:         title,
		Description:   description,
		ScheduledTime: scheduledTime,
		Status:        "scheduled",
		QuestionCount: 0,
	}

	// Сохраняем викторину в БД
	if err := s.quizRepo.Create(quiz); err != nil {
		return nil, fmt.Errorf("failed to create quiz: %w", err)
	}

	return quiz, nil
}

// GetQuizByID возвращает викторину по ID
func (s *QuizService) GetQuizByID(quizID uint) (*entity.Quiz, error) {
	return s.quizRepo.GetByID(quizID)
}

// GetActiveQuiz возвращает активную викторину
func (s *QuizService) GetActiveQuiz() (*entity.Quiz, error) {
	return s.quizRepo.GetActive()
}

// GetScheduledQuizzes возвращает список запланированных викторин
func (s *QuizService) GetScheduledQuizzes() ([]entity.Quiz, error) {
	return s.quizRepo.GetScheduled()
}

// AddQuestions добавляет вопросы к викторине
func (s *QuizService) AddQuestions(quizID uint, questions []entity.Question) error {
	// Получаем викторину, чтобы убедиться, что она существует
	quiz, err := s.quizRepo.GetByID(quizID)
	if err != nil {
		return err
	}

	// Проверяем, что викторина находится в состоянии "scheduled"
	if !quiz.IsScheduled() {
		return errors.New("can only add questions to a scheduled quiz")
	}

	// Получаем существующие вопросы
	existingQuestions, err := s.questionRepo.GetByQuizID(quizID)
	if err != nil {
		return fmt.Errorf("failed to get existing questions: %w", err)
	}

	// Проверяем, не превышает ли общее количество вопросов максимально допустимое
	totalQuestions := len(existingQuestions) + len(questions)
	if totalQuestions > MaxQuizQuestions {
		return fmt.Errorf("максимальное количество вопросов – %d", MaxQuizQuestions)
	}

	// Устанавливаем quizID для всех вопросов
	for i := range questions {
		questions[i].QuizID = quizID
	}

	// Сохраняем вопросы в БД
	if err := s.questionRepo.CreateBatch(questions); err != nil {
		return fmt.Errorf("failed to create questions: %w", err)
	}

	// Обновляем количество вопросов в викторине
	quiz.QuestionCount += len(questions)
	return s.quizRepo.Update(quiz)
}

// ScheduleQuiz планирует время проведения викторины
func (s *QuizService) ScheduleQuiz(quizID uint, scheduledTime time.Time) error {
	// Получаем викторину
	quiz, err := s.quizRepo.GetByID(quizID)
	if err != nil {
		return err
	}

	// Проверяем, что время проведения в будущем
	if scheduledTime.Before(time.Now()) {
		return errors.New("scheduled time must be in the future")
	}

	// Обновляем время проведения
	quiz.ScheduledTime = scheduledTime

	// Если викторина завершена, меняем статус на "scheduled"
	if quiz.IsCompleted() {
		fmt.Printf("[QuizService] Изменение статуса викторины ID=%d с 'completed' на 'scheduled'\n", quizID)
		quiz.Status = "scheduled"
	}

	return s.quizRepo.Update(quiz)
}

// GetQuizWithQuestions возвращает викторину с вопросами
func (s *QuizService) GetQuizWithQuestions(quizID uint) (*entity.Quiz, error) {
	return s.quizRepo.GetWithQuestions(quizID)
}

// ListQuizzes возвращает список викторин с пагинацией
func (s *QuizService) ListQuizzes(page, pageSize int) ([]entity.Quiz, error) {
	offset := (page - 1) * pageSize
	return s.quizRepo.List(pageSize, offset)
}

// DeleteQuiz удаляет викторину
func (s *QuizService) DeleteQuiz(quizID uint) error {
	// Получаем викторину, чтобы убедиться, что она существует
	quiz, err := s.quizRepo.GetByID(quizID)
	if err != nil {
		return err
	}

	// Проверяем, что викторина не активна
	if quiz.IsActive() {
		return errors.New("cannot delete an active quiz")
	}

	return s.quizRepo.Delete(quizID)
}

// GetQuestionsByQuizID возвращает все вопросы для викторины
func (s *QuizService) GetQuestionsByQuizID(quizID uint) ([]entity.Question, error) {
	return s.questionRepo.GetByQuizID(quizID)
}
