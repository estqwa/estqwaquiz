package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	"github.com/yourusername/trivia-api/internal/domain/entity"
	"github.com/yourusername/trivia-api/internal/domain/repository"
	"github.com/yourusername/trivia-api/internal/websocket"
)

// ResultService предоставляет методы для работы с результатами
type ResultService struct {
	resultRepo   repository.ResultRepository
	userRepo     repository.UserRepository
	quizRepo     repository.QuizRepository
	questionRepo repository.QuestionRepository
	cacheRepo    repository.CacheRepository
	db           *gorm.DB
	wsManager    *websocket.Manager
}

// NewResultService создает новый сервис результатов
func NewResultService(
	resultRepo repository.ResultRepository,
	userRepo repository.UserRepository,
	quizRepo repository.QuizRepository,
	questionRepo repository.QuestionRepository,
	cacheRepo repository.CacheRepository,
	db *gorm.DB,
	wsManager *websocket.Manager,
) *ResultService {
	return &ResultService{
		resultRepo:   resultRepo,
		userRepo:     userRepo,
		quizRepo:     quizRepo,
		questionRepo: questionRepo,
		cacheRepo:    cacheRepo,
		db:           db,
		wsManager:    wsManager,
	}
}

/*
// ProcessUserAnswer обрабатывает ответ пользователя на вопрос
// !!! ЭТА ФУНКЦИЯ НЕ ИСПОЛЬЗУЕТСЯ И ЛОГИКА ДУБЛИРУЕТСЯ/РЕАЛИЗОВАНА В quizmanager.AnswerProcessor !!!
// !!! КРОМЕ ТОГО, ЛОГИКА РАСЧЕТА ВРЕМЕНИ ОТВЕТА ЗДЕСЬ НЕКОРРЕКТНА ДЛЯ REAL-TIME !!!
func (s *ResultService) ProcessUserAnswer(userID, quizID, questionID uint, selectedOption int, timestamp int64) (*entity.UserAnswer, error) {
	// Получаем вопрос
	question, err := s.questionRepo.GetByID(questionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get question: %w", err)
	}

	// Проверяем ответ
	isCorrect := question.IsCorrect(selectedOption)
	// Некорректный расчет времени для real-time:
	responseTimeMs := time.Now().UnixMilli() - timestamp // Это время обработки, а не время ответа пользователя

	// Вычисляем очки
	score := question.CalculatePoints(isCorrect, responseTimeMs)

	// TODO: Добавить логику проверки выбывания (elimination), если она нужна здесь
	// isEliminated := !isCorrect // Упрощенный пример

	// Создаем запись об ответе
	userAnswer := &entity.UserAnswer{
		UserID:            userID,
		QuizID:            quizID,
		QuestionID:        questionID,
		SelectedOption:    selectedOption,
		IsCorrect:         isCorrect,
		ResponseTimeMs:    responseTimeMs,
		Score:             score,
		// IsEliminated:      isEliminated,
		// EliminationReason: "",
		CreatedAt:         time.Now(),
	}

	// Сохраняем ответ
	if err := s.resultRepo.SaveUserAnswer(userAnswer); err != nil {
		return nil, fmt.Errorf("failed to save user answer: %w", err)
	}

	return userAnswer, nil
}
*/

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

	// Проверяем статус выбывания из Redis
	eliminationKey := fmt.Sprintf("quiz:%d:eliminated:%d", quizID, userID)
	isEliminated, _ := s.cacheRepo.Exists(eliminationKey)

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
		IsEliminated:   isEliminated,
		CompletedAt:    time.Now(),
	}

	// --- Начало транзакции ---
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Printf("PANIC recovered during CalculateQuizResult transaction: %v", r)
		}
	}()

	if tx.Error != nil {
		log.Printf("Error starting transaction in CalculateQuizResult: %v", tx.Error)
		return nil, tx.Error
	}

	// Сохраняем результат в БД (внутри транзакции)
	if err := tx.Create(result).Error; err != nil {
		tx.Rollback()
		log.Printf("Error saving result in transaction: %v", err)
		return nil, fmt.Errorf("failed to save result: %w", err)
	}

	// Обновляем общий счет пользователя (внутри транзакции)
	if err := tx.Model(&entity.User{}).Where("id = ?", userID).Update("total_score", gorm.Expr("total_score + ?", totalScore)).Error; err != nil {
		tx.Rollback()
		log.Printf("Error updating user score in transaction: %v", err)
		return nil, fmt.Errorf("failed to update user score: %w", err)
	}

	// Обновляем высший счет, если необходимо (внутри транзакции)
	if err := tx.Model(&entity.User{}).Where("id = ? AND highest_score < ?", userID, totalScore).Update("highest_score", totalScore).Error; err != nil {
		// Не откатываем транзакцию из-за этой ошибки, она не критична
		log.Printf("Warning: Error updating user highest score: %v", err)
	}

	// Увеличиваем счетчик сыгранных игр (внутри транзакции)
	if err := tx.Model(&entity.User{}).Where("id = ?", userID).UpdateColumn("games_played", gorm.Expr("games_played + ?", 1)).Error; err != nil {
		tx.Rollback()
		log.Printf("Error incrementing games played in transaction: %v", err)
		return nil, fmt.Errorf("failed to increment games played: %w", err)
	}

	// --- Коммит транзакции ---
	if err := tx.Commit().Error; err != nil {
		log.Printf("Error committing transaction in CalculateQuizResult: %v", err)
		return nil, err
	}

	log.Printf("[ResultService] Успешно рассчитан и сохранен результат для пользователя #%d в викторине #%d", userID, quizID)
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

// DetermineWinnersAndAllocatePrizes финализирует результаты викторины,
// вызывая расчет рангов и призов в репозитории.
// Предполагается, что базовые результаты (Score, CorrectAnswers, IsEliminated)
// уже сохранены в таблице results для каждого участника.
func (s *ResultService) DetermineWinnersAndAllocatePrizes(ctx context.Context, quizID uint) error {
	log.Printf("[ResultService] Финализация результатов для викторины #%d", quizID)

	// 1. Вызываем CalculateRanks для расчета рангов, определения победителей и призов в БД
	// Эта функция теперь является единственным источником правды для этих данных.
	if err := s.resultRepo.CalculateRanks(quizID); err != nil {
		log.Printf("[ResultService] Ошибка при расчете рангов для викторины #%d: %v", quizID, err)
		// В зависимости от логики, можно либо вернуть ошибку, либо продолжить
		// Например, если обновление статуса и отправка WS важны даже при ошибке рангов.
		return fmt.Errorf("ошибка расчета рангов: %w", err)
	}
	log.Printf("[ResultService] Ранги и призы для викторины #%d успешно рассчитаны и сохранены.", quizID)

	// 2. (Опционально) Обновляем статус викторины на "завершена"
	// TODO: Добавить обновление статуса викторины в `quizRepo`, если необходимо
	// if err := s.quizRepo.UpdateStatus(quizID, "completed"); err != nil {
	// 	 log.Printf("[ResultService] Ошибка при обновлении статуса викторины #%d на 'completed': %v", quizID, err)
	//   // Рассмотреть, является ли эта ошибка критичной
	// }

	// ===>>> ИЗМЕНЕНИЕ: ОТПРАВКА УВЕДОМЛЕНИЯ О ДОСТУПНОСТИ РЕЗУЛЬТАТОВ <<<===
	if s.wsManager != nil {
		resultsAvailableEvent := map[string]interface{}{
			"quiz_id": quizID,
		}
		fullEvent := map[string]interface{}{ // Используем стандартную структуру события
			"type": "quiz:results_available",
			"data": resultsAvailableEvent,
		}
		if err := s.wsManager.BroadcastEventToQuiz(quizID, fullEvent); err != nil {
			// Логируем ошибку, но не прерываем выполнение, т.к. основная работа сделана
			log.Printf("[ResultService] Ошибка при отправке события quiz:results_available для викторины #%d: %v", quizID, err)
		} else {
			log.Printf("[ResultService] Событие quiz:results_available для викторины #%d успешно отправлено", quizID)
		}
	} else {
		log.Println("[ResultService] Менеджер WebSocket не инициализирован, уведомление quiz:results_available не отправлено.")
	}
	// ===>>> КОНЕЦ ИЗМЕНЕНИЯ <<<===

	log.Printf("[ResultService] Финализация результатов для викторины #%d успешно завершена.", quizID)
	return nil
}

// GetQuizWinners возвращает список победителей викторины
func (s *ResultService) GetQuizWinners(quizID uint) ([]entity.Result, error) {
	return s.resultRepo.GetQuizWinners(quizID)
}
