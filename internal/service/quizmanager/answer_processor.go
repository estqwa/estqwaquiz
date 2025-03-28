package quizmanager

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/yourusername/trivia-api/internal/domain/entity"
)

// AnswerProcessor отвечает за обработку ответов пользователей
type AnswerProcessor struct {
	// Настройки
	config *Config

	// Зависимости
	deps *Dependencies
}

// NewAnswerProcessor создает новый процессор ответов
func NewAnswerProcessor(config *Config, deps *Dependencies) *AnswerProcessor {
	return &AnswerProcessor{
		config: config,
		deps:   deps,
	}
}

// ProcessAnswer обрабатывает ответ пользователя
func (ap *AnswerProcessor) ProcessAnswer(
	ctx context.Context,
	userID uint,
	questionID uint,
	selectedOption int,
	timestamp int64,
	quizState *ActiveQuizState,
) error {
	log.Printf("[AnswerProcessor] Обработка ответа пользователя #%d на вопрос #%d, выбранный вариант: %d",
		userID, questionID, selectedOption)

	// Проверяем наличие активной викторины
	if quizState == nil || quizState.Quiz == nil {
		log.Printf("[AnswerProcessor] Ошибка: нет активной викторины для ответа пользователя #%d", userID)
		return fmt.Errorf("no active quiz")
	}

	// Получаем текущий вопрос из состояния
	currentQuestion, _ := quizState.GetCurrentQuestion()
	if currentQuestion == nil || currentQuestion.ID != questionID {
		log.Printf("[AnswerProcessor] Ошибка: вопрос #%d не является текущим активным вопросом", questionID)
		return fmt.Errorf("question is not the current active question")
	}

	// Создаем ключ для Redis для проверки статуса пользователя
	userStatusKey := fmt.Sprintf("quiz:%d:user:%d:status", quizState.Quiz.ID, userID)

	// Проверяем, не выбыл ли пользователь уже
	userStatus, _ := ap.deps.CacheRepo.Get(userStatusKey)
	if userStatus == "eliminated" {
		log.Printf("[AnswerProcessor] Пользователь #%d уже выбыл из викторины и не может отвечать", userID)

		// Отправляем напоминание пользователю, что он выбыл
		eliminationReminder := map[string]interface{}{
			"message":     "Вы выбыли из викторины и можете только наблюдать",
			"question_id": questionID,
		}

		if err := ap.deps.WSManager.SendEventToUser(fmt.Sprintf("%d", userID), "quiz:elimination_reminder", eliminationReminder); err != nil {
			log.Printf("[AnswerProcessor] ОШИБКА при отправке напоминания о выбывании пользователю #%d: %v", userID, err)
		}

		return fmt.Errorf("user is already eliminated from the quiz")
	}

	// Получаем время начала вопроса из кеша
	questionStartKey := fmt.Sprintf("question:%d:start_time", questionID)
	startTimeStr, err := ap.deps.CacheRepo.Get(questionStartKey)
	if err != nil {
		log.Printf("[AnswerProcessor] Ошибка при получении времени начала вопроса #%d: %v", questionID, err)
		return fmt.Errorf("question start time not found: %w", err)
	}

	startTime, err := strconv.ParseInt(startTimeStr, 10, 64)
	if err != nil {
		log.Printf("[AnswerProcessor] Ошибка при парсинге времени начала вопроса #%d: %v", questionID, err)
		return fmt.Errorf("invalid question start time: %w", err)
	}

	// Вычисляем время ответа
	responseTimeMs := timestamp - startTime

	// Проверяем, что время ответа не превышает лимит
	timeLimit := int64(currentQuestion.TimeLimitSec * 1000)
	isTimeLimitExceeded := responseTimeMs > timeLimit

	// Проверяем, выбывает ли пользователь из-за слишком долгого ответа
	isCriticalTimeExceeded := responseTimeMs > ap.config.EliminationTimeMs

	// Проверяем, правильный ли ответ
	isCorrect := currentQuestion.IsCorrect(selectedOption)
	correctOption := currentQuestion.CorrectOption

	// Вычисляем количество очков
	score := currentQuestion.CalculatePoints(isCorrect, responseTimeMs)

	// Проверяем, нужно ли выбывать пользователю (неверный ответ или слишком долгий ответ)
	isEliminated := !isCorrect || isCriticalTimeExceeded

	if isEliminated {
		reason := "неверный ответ"
		if isCriticalTimeExceeded {
			reason = fmt.Sprintf("превышено время ответа (>%d сек)", ap.config.EliminationTimeMs/1000)
		}

		log.Printf("[AnswerProcessor] Пользователь #%d выбывает из викторины. Причина: %s", userID, reason)

		// Устанавливаем статус пользователя как "выбывший"
		if err := ap.deps.CacheRepo.Set(userStatusKey, "eliminated", 24*time.Hour); err != nil {
			log.Printf("[AnswerProcessor] Ошибка при установке статуса выбывшего пользователя #%d: %v", userID, err)
		}
	}

	// Создаем запись об ответе
	userAnswer := &entity.UserAnswer{
		UserID:         userID,
		QuizID:         quizState.Quiz.ID,
		QuestionID:     questionID,
		SelectedOption: selectedOption,
		IsCorrect:      isCorrect,
		ResponseTimeMs: responseTimeMs,
		Score:          score,
	}

	// Сохраняем ответ в БД
	if err := ap.deps.ResultRepo.SaveUserAnswer(userAnswer); err != nil {
		log.Printf("[AnswerProcessor] Ошибка при сохранении ответа пользователя #%d на вопрос #%d: %v",
			userID, questionID, err)
		return fmt.Errorf("failed to save user answer: %w", err)
	}

	// Отправляем результат пользователю
	answerResultEvent := map[string]interface{}{
		"question_id":         questionID,
		"correct_option":      correctOption,
		"your_answer":         selectedOption,
		"is_correct":          isCorrect,
		"points_earned":       score,
		"time_taken_ms":       responseTimeMs,
		"is_eliminated":       isEliminated,
		"time_limit_exceeded": isTimeLimitExceeded,
	}

	if err := ap.deps.WSManager.SendEventToUser(fmt.Sprintf("%d", userID), "quiz:answer_result", answerResultEvent); err != nil {
		log.Printf("[AnswerProcessor] Ошибка при отправке результата ответа пользователю #%d: %v", userID, err)
		// Не возвращаем ошибку, так как ответ уже сохранен в БД
	} else {
		log.Printf("[AnswerProcessor] Успешно обработан ответ пользователя #%d на вопрос #%d: %v, очков: %d, время: %d мс",
			userID, questionID, isCorrect, score, responseTimeMs)

		// Если пользователь выбыл, отправляем дополнительное сообщение
		if isEliminated {
			reason := "неверный ответ"
			if isCriticalTimeExceeded {
				reason = fmt.Sprintf("превышено время ответа (>%d сек)", ap.config.EliminationTimeMs/1000)
			}

			eliminationEvent := map[string]interface{}{
				"message": "Вы выбыли из викторины и можете только наблюдать",
				"reason":  reason,
			}

			if err := ap.deps.WSManager.SendEventToUser(fmt.Sprintf("%d", userID), "quiz:elimination", eliminationEvent); err != nil {
				log.Printf("[AnswerProcessor] Ошибка при отправке уведомления о выбывании пользователю #%d: %v", userID, err)
			}
		}
	}

	return nil
}

// HandleReadyEvent обрабатывает событие готовности пользователя
func (ap *AnswerProcessor) HandleReadyEvent(ctx context.Context, userID uint, quizID uint) error {
	log.Printf("[AnswerProcessor] Пользователь #%d отметился как готовый к викторине #%d", userID, quizID)

	// Создаем ключ для Redis и сохраняем информацию о готовности
	readyKey := fmt.Sprintf("quiz:%d:ready_users", quizID)
	userReadyKey := fmt.Sprintf("%s:%d", readyKey, userID)

	if err := ap.deps.CacheRepo.Set(userReadyKey, "1", time.Hour); err != nil {
		log.Printf("[AnswerProcessor] Ошибка при сохранении готовности пользователя #%d к викторине #%d: %v",
			userID, quizID, err)
		return fmt.Errorf("failed to save ready status: %w", err)
	}

	// Отправляем информацию о готовности пользователя всем участникам
	readyEvent := map[string]interface{}{
		"user_id": userID,
		"quiz_id": quizID,
		"status":  "ready",
	}

	if err := ap.deps.WSManager.BroadcastEvent("quiz:user_ready", readyEvent); err != nil {
		log.Printf("[AnswerProcessor] Ошибка при отправке события готовности пользователя #%d к викторине #%d: %v",
			userID, quizID, err)
		return fmt.Errorf("failed to broadcast ready event: %w", err)
	}

	log.Printf("[AnswerProcessor] Успешно отправлено событие о готовности пользователя #%d к викторине #%d",
		userID, quizID)

	return nil
}
