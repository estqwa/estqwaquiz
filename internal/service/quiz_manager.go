package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/yourusername/trivia-api/internal/domain/entity"
	"github.com/yourusername/trivia-api/internal/domain/repository"
	"github.com/yourusername/trivia-api/internal/service/quizmanager"
	"github.com/yourusername/trivia-api/internal/websocket"
)

// QuizManager координирует работу компонентов для управления викторинами
type QuizManager struct {
	// Компоненты системы
	scheduler       *quizmanager.Scheduler
	questionManager *quizmanager.QuestionManager
	answerProcessor *quizmanager.AnswerProcessor

	// Репозитории для прямого доступа
	quizRepo      repository.QuizRepository
	resultService *ResultService

	// Состояние активной викторины
	activeQuizState *quizmanager.ActiveQuizState

	// Контекст для управления жизненным циклом
	ctx    context.Context
	cancel context.CancelFunc

	// Зависимости
	deps *quizmanager.Dependencies
}

// NewQuizManager создает новый экземпляр менеджера викторин
func NewQuizManager(
	quizRepo repository.QuizRepository,
	questionRepo repository.QuestionRepository,
	resultRepo repository.ResultRepository,
	resultService *ResultService,
	cacheRepo repository.CacheRepository,
	wsManager *websocket.Manager,
) *QuizManager {
	// Создаем контекст для управления жизненным циклом
	ctx, cancel := context.WithCancel(context.Background())

	// Создаем конфигурацию
	config := quizmanager.DefaultConfig()

	// Собираем зависимости
	deps := &quizmanager.Dependencies{
		QuizRepo:      quizRepo,
		QuestionRepo:  questionRepo,
		ResultRepo:    resultRepo,
		ResultService: resultService,
		CacheRepo:     cacheRepo,
		WSManager:     wsManager,
	}

	// Создаем компоненты
	scheduler := quizmanager.NewScheduler(config, deps)
	questionManager := quizmanager.NewQuestionManager(config, deps)
	answerProcessor := quizmanager.NewAnswerProcessor(config, deps)

	qm := &QuizManager{
		scheduler:       scheduler,
		questionManager: questionManager,
		answerProcessor: answerProcessor,
		quizRepo:        quizRepo,
		resultService:   resultService,
		ctx:             ctx,
		cancel:          cancel,
		deps:            deps,
	}

	// Запускаем слушателя событий
	go qm.handleEvents()

	log.Println("[QuizManager] Менеджер викторин успешно инициализирован")
	return qm
}

// handleEvents обрабатывает события от компонентов
func (qm *QuizManager) handleEvents() {
	// Слушаем события запуска викторин
	quizStartCh := qm.scheduler.GetQuizStartChannel()
	// Слушаем события завершения вопросов
	questionDoneCh := qm.questionManager.QuestionDone()

	for {
		select {
		case <-qm.ctx.Done():
			log.Println("[QuizManager] Завершение работы слушателя событий")
			return

		case quizID := <-quizStartCh:
			// Обрабатываем событие запуска викторины
			go qm.handleQuizStart(quizID)

		case <-questionDoneCh:
			// Обрабатываем событие завершения вопросов
			if qm.activeQuizState != nil && qm.activeQuizState.Quiz != nil {
				go qm.finishQuiz(qm.activeQuizState.Quiz.ID)
			}
		}
	}
}

// ScheduleQuiz планирует запуск викторины в указанное время
func (qm *QuizManager) ScheduleQuiz(quizID uint, scheduledTime time.Time) error {
	log.Printf("[QuizManager] Планирование викторины #%d на %v", quizID, scheduledTime)
	return qm.scheduler.ScheduleQuiz(qm.ctx, quizID, scheduledTime)
}

// CancelQuiz отменяет запланированную викторину
func (qm *QuizManager) CancelQuiz(quizID uint) error {
	log.Printf("[QuizManager] Отмена викторины #%d", quizID)
	return qm.scheduler.CancelQuiz(quizID)
}

// handleQuizStart обрабатывает запуск викторины
func (qm *QuizManager) handleQuizStart(quizID uint) {
	log.Printf("[QuizManager] Обработка запуска викторины #%d", quizID)

	// Получаем викторину с вопросами
	quiz, err := qm.quizRepo.GetWithQuestions(quizID)
	if err != nil {
		log.Printf("[QuizManager] Ошибка при получении викторины #%d: %v", quizID, err)
		return
	}

	// Убеждаемся, что у викторины есть вопросы
	if len(quiz.Questions) == 0 {
		log.Printf("[QuizManager] Викторина #%d не имеет вопросов, запуск отменён", quizID)
		return
	}

	// Создаем состояние активной викторины
	qm.activeQuizState = quizmanager.NewActiveQuizState(quiz)

	// Запускаем процесс отправки вопросов
	go func() {
		if err := qm.questionManager.RunQuizQuestions(qm.ctx, qm.activeQuizState); err != nil {
			log.Printf("[QuizManager] Ошибка при выполнении викторины #%d: %v", quizID, err)
		}
	}()
}

// finishQuiz завершает викторину и подсчитывает результаты
func (qm *QuizManager) finishQuiz(quizID uint) {
	log.Printf("[QuizManager] Завершение викторины #%d", quizID)

	if qm.activeQuizState == nil || qm.activeQuizState.Quiz == nil || qm.activeQuizState.Quiz.ID != quizID {
		log.Printf("[QuizManager] Ошибка: викторина #%d не является активной", quizID)
		return
	}

	// Обновляем статус викторины
	quiz := qm.activeQuizState.Quiz
	quiz.Status = "completed"
	// Для timestamp завершения используем текущее время
	completedAt := time.Now()

	if err := qm.quizRepo.Update(quiz); err != nil {
		log.Printf("[QuizManager] Ошибка при обновлении статуса викторины #%d: %v", quizID, err)
		// Продолжаем несмотря на ошибку
	}

	// Отправляем событие о завершении
	finishEvent := map[string]interface{}{
		"quiz_id":  quizID,
		"title":    quiz.Title,
		"message":  "Викторина завершена! Подсчет результатов...",
		"status":   "completed",
		"ended_at": completedAt,
	}

	// Отправляем всем участникам через WebSocket-менеджер
	wsManager := qm.deps.WSManager
	if err := wsManager.BroadcastEvent("quiz:finish", finishEvent); err != nil {
		log.Printf("[QuizManager] Ошибка при отправке события о завершении викторины #%d: %v", quizID, err)
	}

	// Сбрасываем активную викторину
	qm.activeQuizState = nil

	// Подсчитываем и отправляем результаты (асинхронно)
	go qm.calculateAndSendResults(quizID)
}

// calculateAndSendResults подсчитывает и отправляет результаты викторины
func (qm *QuizManager) calculateAndSendResults(quizID uint) {
	log.Printf("[QuizManager] Подсчет и отправка результатов для викторины #%d", quizID)

	// Здесь могла бы быть логика получения участников викторины
	// и вычисления результатов для каждого

	// Пока просто отправляем событие о том, что результаты доступны
	resultsAvailableEvent := map[string]interface{}{
		"quiz_id": quizID,
		"message": "Результаты викторины доступны",
	}

	// Отправляем событие всем пользователям через WebSocket-менеджер
	if err := qm.deps.WSManager.BroadcastEvent("quiz:results_available", resultsAvailableEvent); err != nil {
		log.Printf("[QuizManager] Ошибка при отправке события о доступности результатов: %v", err)
	}
}

// ProcessAnswer обрабатывает ответ пользователя на вопрос
func (qm *QuizManager) ProcessAnswer(userID, questionID uint, selectedOption int, timestamp int64) error {
	if qm.activeQuizState == nil {
		return fmt.Errorf("нет активной викторины")
	}

	return qm.answerProcessor.ProcessAnswer(
		qm.ctx, userID, questionID, selectedOption, timestamp, qm.activeQuizState)
}

// HandleReadyEvent обрабатывает событие готовности пользователя
func (qm *QuizManager) HandleReadyEvent(userID uint, quizID uint) error {
	return qm.answerProcessor.HandleReadyEvent(qm.ctx, userID, quizID)
}

// GetActiveQuiz возвращает активную викторину
func (qm *QuizManager) GetActiveQuiz() *entity.Quiz {
	if qm.activeQuizState == nil {
		return nil
	}
	return qm.activeQuizState.Quiz
}

// AutoFillQuizQuestions автоматически заполняет викторину вопросами
func (qm *QuizManager) AutoFillQuizQuestions(quizID uint) error {
	return qm.questionManager.AutoFillQuizQuestions(qm.ctx, quizID)
}

// Shutdown корректно завершает работу менеджера викторин
func (qm *QuizManager) Shutdown() {
	log.Println("[QuizManager] Завершение работы менеджера викторин...")

	// Отменяем контекст для завершения всех операций
	qm.cancel()

	// Здесь могли бы быть дополнительные действия по завершению работы

	log.Println("[QuizManager] Менеджер викторин остановлен")
}
