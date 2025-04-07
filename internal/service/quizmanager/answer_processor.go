package quizmanager

import (
	"context"
	"fmt"
	"log"
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

	quizID := quizState.Quiz.ID

	// -------------------- Начало проверок выбывания и дубликатов --------------------
	// Проверяем, не выбыл ли пользователь
	eliminationKey := fmt.Sprintf("quiz:%d:eliminated:%d", quizID, userID)
	isEliminated, _ := ap.deps.CacheRepo.Exists(eliminationKey)
	if isEliminated {
		log.Printf("[AnswerProcessor] Пользователь #%d уже выбыл из викторины #%d", userID, quizID)
		// Можно отправить повторное уведомление, если нужно
		// ap.sendEliminationNotification(userID, quizID, "already_eliminated")
		return fmt.Errorf("user is eliminated from this quiz")
	}

	// ===>>> ИЗМЕНЕНИЕ: Используем атомарный SetNX вместо Exists + Set <<<===
	// Пытаемся установить флаг, что пользователь ответил на этот вопрос.
	// SetNX вернет true, если ключ УСПЕШНО установлен (т.е. его не было).
	answerKey := fmt.Sprintf("quiz:%d:user:%d:question:%d", quizID, userID, questionID)
	wasSet, err := ap.deps.CacheRepo.SetNX(answerKey, "1", 1*time.Hour)

	// Логируем ошибку Redis, но не обязательно прерываем выполнение, если не критично
	if err != nil {
		log.Printf("[AnswerProcessor] WARNING: Ошибка Redis при попытке SetNX для user #%d, question #%d: %v", userID, questionID, err)
		// Можно рассмотреть возврат ошибки, если надежность Redis критична
		// return fmt.Errorf("redis error during answer check: %w", err)
	}

	// Если ключ НЕ был установлен (wasSet == false), значит он уже существовал.
	if !wasSet {
		log.Printf("[AnswerProcessor] Пользователь #%d уже отвечал на вопрос #%d викторины #%d (определено через SetNX)", userID, questionID, quizID)
		return fmt.Errorf("user already answered this question")
	}
	// ===>>> КОНЕЦ ИЗМЕНЕНИЯ <<<===

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

	// Получаем время начала вопроса из состояния викторины
	startTime := quizState.GetCurrentQuestionStartTime()
	if startTime == 0 {
		// Обработка случая, если время старта не установлено (ошибка логики)
		log.Printf("[AnswerProcessor] CRITICAL: Время начала для вопроса #%d не найдено в состоянии викторины #%d", questionID, quizID)
		return fmt.Errorf("internal error: question start time not found in state")
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
	userShouldBeEliminated := !isCorrect || responseTimeMs > timeLimit
	eliminationReason := ""
	if userShouldBeEliminated {
		if !isCorrect {
			eliminationReason = "incorrect_answer"
		} else {
			eliminationReason = "time_exceeded"
		}

		log.Printf("[AnswerProcessor] Пользователь #%d выбывает из викторины #%d. Причина: %s", userID, quizID, eliminationReason)

		// Устанавливаем статус пользователя как "выбывший" в Redis
		// Логируем ошибку, но не прерываем основной поток
		if err := ap.deps.CacheRepo.Set(eliminationKey, "1", 24*time.Hour); err != nil {
			log.Printf("[AnswerProcessor] WARNING: Не удалось установить статус выбывшего пользователя #%d в Redis: %v", userID, err)
		}

		// Отправляем уведомление о выбывании пользователю
		ap.sendEliminationNotification(userID, quizID, eliminationReason)
	}

	// Создаем запись об ответе
	userAnswer := &entity.UserAnswer{
		UserID:            userID,
		QuizID:            quizID,
		QuestionID:        questionID,
		SelectedOption:    selectedOption,
		IsCorrect:         isCorrect,
		ResponseTimeMs:    responseTimeMs,
		Score:             score,
		IsEliminated:      userShouldBeEliminated, // Сохраняем статус выбывания в ответе
		EliminationReason: eliminationReason,      // Сохраняем причину
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
		"is_eliminated":       userShouldBeEliminated,
		"time_limit_exceeded": isTimeLimitExceeded,
	}

	if err := ap.deps.WSManager.SendEventToUser(fmt.Sprintf("%d", userID), "quiz:answer_result", answerResultEvent); err != nil {
		log.Printf("[AnswerProcessor] Ошибка при отправке результата ответа пользователю #%d: %v", userID, err)
		// Не возвращаем ошибку, так как ответ уже сохранен в БД
	} else {
		log.Printf("[AnswerProcessor] Успешно обработан ответ пользователя #%d на вопрос #%d: %v, очков: %d, время: %d мс",
			userID, questionID, isCorrect, score, responseTimeMs)

		// Если пользователь выбыл, отправляем дополнительное сообщение
		if userShouldBeEliminated {
			reason := eliminationReason
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
	fullEvent := map[string]interface{}{
		"type": "quiz:user_ready",
		"data": map[string]interface{}{
			"user_id": userID,
			"quiz_id": quizID,
			"status":  "ready",
		},
	}

	if err := ap.deps.WSManager.BroadcastEventToQuiz(quizID, fullEvent); err != nil {
		log.Printf("[AnswerProcessor] Ошибка при отправке события готовности пользователя #%d к викторине #%d: %v",
			userID, quizID, err)
		return fmt.Errorf("failed to broadcast ready event: %w", err)
	}

	log.Printf("[AnswerProcessor] Успешно отправлено событие о готовности пользователя #%d к викторине #%d",
		userID, quizID)

	return nil
}

// Новый вспомогательный метод для отправки уведомления о выбывании
func (ap *AnswerProcessor) sendEliminationNotification(userID uint, quizID uint, reason string) {
	eliminationEvent := map[string]interface{}{
		"quiz_id": quizID,
		"user_id": userID, // Включаем UserID, чтобы клиент мог это проверить
		"reason":  reason,
		"message": "Вы выбыли из викторины и можете только наблюдать",
	}

	if err := ap.deps.WSManager.SendEventToUser(fmt.Sprintf("%d", userID), "quiz:elimination", eliminationEvent); err != nil {
		log.Printf("[AnswerProcessor] Ошибка при отправке уведомления о выбывании пользователю #%d: %v", userID, err)
	}
}
