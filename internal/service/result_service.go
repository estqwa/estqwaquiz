package service

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/yourusername/trivia-api/internal/domain/entity"
	"github.com/yourusername/trivia-api/internal/domain/repository"
)

// ResultService предоставляет методы для работы с результатами
type ResultService struct {
	resultRepo   repository.ResultRepository
	userRepo     repository.UserRepository
	quizRepo     repository.QuizRepository
	questionRepo repository.QuestionRepository
	cacheRepo    repository.CacheRepository
}

// NewResultService создает новый сервис результатов
func NewResultService(
	resultRepo repository.ResultRepository,
	userRepo repository.UserRepository,
	quizRepo repository.QuizRepository,
	questionRepo repository.QuestionRepository,
	cacheRepo repository.CacheRepository,
) *ResultService {
	return &ResultService{
		resultRepo:   resultRepo,
		userRepo:     userRepo,
		quizRepo:     quizRepo,
		questionRepo: questionRepo,
		cacheRepo:    cacheRepo,
	}
}

// ProcessUserAnswer обрабатывает ответ пользователя на вопрос
// Примечание: основное использование этого метода для случаев,
// когда нет активной викторины под управлением QuizManager
func (s *ResultService) ProcessUserAnswer(userID, quizID, questionID uint, selectedOption int, timestamp int64) (*entity.UserAnswer, error) {
	log.Printf("[ResultService] Обработка ответа пользователя #%d на вопрос #%d, выбранный вариант: %d",
		userID, questionID, selectedOption)

	// Получаем вопрос
	question, err := s.questionRepo.GetByID(questionID)
	if err != nil {
		log.Printf("[ResultService] Ошибка при получении вопроса #%d: %v", questionID, err)
		return nil, fmt.Errorf("failed to get question: %w", err)
	}

	// Проверяем, что вопрос относится к указанной викторине
	if question.QuizID != quizID {
		log.Printf("[ResultService] Ошибка: вопрос #%d относится к викторине #%d, а не к запрошенной викторине #%d",
			questionID, question.QuizID, quizID)
		return nil, errors.New("question does not belong to the specified quiz")
	}

	// Получаем время начала вопроса из кеша
	questionStartKey := fmt.Sprintf("question:%d:start_time", questionID)
	startTimeStr, err := s.cacheRepo.Get(questionStartKey)
	if err != nil {
		log.Printf("[ResultService] Ошибка при получении времени начала вопроса #%d: %v", questionID, err)
		return nil, fmt.Errorf("question start time not found: %w", err)
	}

	startTime, err := strconv.ParseInt(startTimeStr, 10, 64)
	if err != nil {
		log.Printf("[ResultService] Ошибка при парсинге времени начала вопроса #%d: %v", questionID, err)
		return nil, fmt.Errorf("invalid question start time: %w", err)
	}

	// Вычисляем время ответа
	responseTimeMs := timestamp - startTime

	// Проверяем, что время ответа не превышает лимит
	if responseTimeMs > int64(question.TimeLimitSec*1000) {
		log.Printf("[ResultService] Ошибка: превышено время ответа пользователя #%d на вопрос #%d (%d мс > %d мс)",
			userID, questionID, responseTimeMs, question.TimeLimitSec*1000)
		return nil, errors.New("time limit exceeded")
	}

	// Проверяем, правильный ли ответ
	isCorrect := question.IsCorrect(selectedOption)

	// Вычисляем количество очков
	score := question.CalculatePoints(isCorrect, responseTimeMs)

	// Создаем запись об ответе
	userAnswer := &entity.UserAnswer{
		UserID:         userID,
		QuizID:         quizID,
		QuestionID:     questionID,
		SelectedOption: selectedOption,
		IsCorrect:      isCorrect,
		ResponseTimeMs: responseTimeMs,
		Score:          score,
	}

	// Сохраняем ответ в БД
	if err := s.resultRepo.SaveUserAnswer(userAnswer); err != nil {
		return nil, fmt.Errorf("failed to save user answer: %w", err)
	}

	return userAnswer, nil
}

// CalculateQuizResult подсчитывает итоговый результат пользователя в викторине
func (s *ResultService) CalculateQuizResult(userID, quizID uint) (*entity.Result, error) {
	// Получаем информацию о пользователе
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, err
	}

	// Получаем информацию о викторине
	quiz, err := s.quizRepo.GetWithQuestions(quizID)
	if err != nil {
		return nil, err
	}

	// Получаем все ответы пользователя
	userAnswers, err := s.resultRepo.GetUserAnswers(userID, quizID)
	if err != nil {
		return nil, err
	}

	// Подсчитываем общий счет и количество правильных ответов
	totalScore := 0
	correctAnswers := 0
	for _, answer := range userAnswers {
		totalScore += answer.Score
		if answer.IsCorrect {
			correctAnswers++
		}
	}

	// Создаем запись о результате
	result := &entity.Result{
		UserID:         userID,
		QuizID:         quizID,
		Username:       user.Username,
		ProfilePicture: user.ProfilePicture,
		Score:          totalScore,
		CorrectAnswers: correctAnswers,
		TotalQuestions: len(quiz.Questions),
		CompletedAt:    time.Now(),
	}

	// Сохраняем результат в БД
	if err := s.resultRepo.SaveResult(result); err != nil {
		return nil, fmt.Errorf("failed to save result: %w", err)
	}

	// Обновляем общий счет пользователя
	if err := s.userRepo.UpdateScore(userID, totalScore); err != nil {
		return nil, fmt.Errorf("failed to update user score: %w", err)
	}

	// Увеличиваем счетчик сыгранных игр
	if err := s.userRepo.IncrementGamesPlayed(userID); err != nil {
		return nil, fmt.Errorf("failed to increment games played: %w", err)
	}

	return result, nil
}

// GetQuizResults возвращает все результаты для викторины
func (s *ResultService) GetQuizResults(quizID uint) ([]entity.Result, error) {
	// Пересчитываем ранги перед получением результатов
	if err := s.resultRepo.CalculateRanks(quizID); err != nil {
		return nil, err
	}

	return s.resultRepo.GetQuizResults(quizID)
}

// GetUserResult возвращает результат пользователя для конкретной викторины
func (s *ResultService) GetUserResult(userID, quizID uint) (*entity.Result, error) {
	return s.resultRepo.GetUserResult(userID, quizID)
}

// GetUserResults возвращает все результаты пользователя с пагинацией
func (s *ResultService) GetUserResults(userID uint, page, pageSize int) ([]entity.Result, error) {
	offset := (page - 1) * pageSize
	return s.resultRepo.GetUserResults(userID, pageSize, offset)
}
