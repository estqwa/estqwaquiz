package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/yourusername/trivia-api/internal/domain/entity"
	"github.com/yourusername/trivia-api/internal/domain/repository"
	"github.com/yourusername/trivia-api/internal/service/quizmanager"
	"github.com/yourusername/trivia-api/internal/websocket"
	"gorm.io/gorm"
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
	wsManager     *websocket.Manager

	// Состояние активной викторины
	activeQuizState *quizmanager.ActiveQuizState
	stateMutex      sync.RWMutex

	// Контекст для управления жизненным циклом
	ctx    context.Context
	cancel context.CancelFunc

	// Зависимости
	// deps *quizmanager.Dependencies
}

// NewQuizManager создает новый экземпляр менеджера викторин
func NewQuizManager(
	quizRepo repository.QuizRepository,
	questionRepo repository.QuestionRepository,
	resultRepo repository.ResultRepository,
	resultService *ResultService,
	cacheRepo repository.CacheRepository,
	wsManager *websocket.Manager,
	db *gorm.DB,
) *QuizManager {
	// Создаем контекст для управления жизненным циклом
	ctx, cancel := context.WithCancel(context.Background())

	// Создаем конфигурацию
	config := quizmanager.DefaultConfig()

	// Собираем зависимости для компонентов
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
		wsManager:       wsManager,
		ctx:             ctx,
		cancel:          cancel,
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
			qm.stateMutex.RLock()
			activeState := qm.activeQuizState
			qm.stateMutex.RUnlock()
			if activeState != nil && activeState.Quiz != nil {
				go qm.finishQuiz(activeState.Quiz.ID)
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
	newState := quizmanager.NewActiveQuizState(quiz)

	// Блокируем для записи
	qm.stateMutex.Lock()
	// Проверяем, не запущена ли уже другая викторина (на всякий случай)
	if qm.activeQuizState != nil {
		log.Printf("[QuizManager] WARNING: Попытка запустить викторину #%d, когда викторина #%d уже активна!", quizID, qm.activeQuizState.Quiz.ID)
		qm.stateMutex.Unlock()
		return
	}
	qm.activeQuizState = newState
	qm.stateMutex.Unlock()

	// Запускаем процесс отправки вопросов
	go func() {
		if err := qm.questionManager.RunQuizQuestions(qm.ctx, newState); err != nil {
			log.Printf("[QuizManager] Ошибка при выполнении викторины #%d: %v", quizID, err)
			// В случае ошибки выполнения, также завершаем викторину
			qm.finishQuiz(quizID)
		}
	}()
}

// finishQuiz завершает викторину и подсчитывает результаты
func (qm *QuizManager) finishQuiz(quizID uint) {
	log.Printf("[QuizManager] Завершение викторины #%d", quizID)

	// Блокируем для чтения и записи
	qm.stateMutex.Lock()
	defer qm.stateMutex.Unlock() // Гарантируем разблокировку

	if qm.activeQuizState == nil || qm.activeQuizState.Quiz == nil || qm.activeQuizState.Quiz.ID != quizID {
		log.Printf("[QuizManager] Ошибка: викторина #%d не является активной или уже завершена.", quizID)
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
	// if err := qm.wsManager.BroadcastEventToQuiz(quizID, "quiz:finish", finishEvent); err != nil {
	// Используем новую сигнатуру
	fullEvent := map[string]interface{}{ // Или websocket.Event
		"type": "quiz:finish",
		"data": finishEvent,
	}
	if err := qm.wsManager.BroadcastEventToQuiz(quizID, fullEvent); err != nil {
		log.Printf("[QuizManager] Ошибка при отправке события о завершении викторины #%d: %v", quizID, err)
	}

	// --- Вызов определения победителей ---
	// Запускаем асинхронно, чтобы не блокировать завершение викторины
	go func(ctx context.Context, currentQuizID uint) {
		// Даем небольшую задержку, чтобы убедиться, что все последние события обработаны
		time.Sleep(2 * time.Second)
		if err := qm.resultService.DetermineWinnersAndAllocatePrizes(ctx, currentQuizID); err != nil {
			log.Printf("[QuizManager] Ошибка при определении победителей для викторины #%d: %v", currentQuizID, err)
		}
	}(qm.ctx, quizID) // Передаем quizID в горутину

	// Сбрасываем активную викторину
	qm.activeQuizState = nil
}

// ProcessAnswer обрабатывает ответ пользователя на вопрос
func (qm *QuizManager) ProcessAnswer(userID, questionID uint, selectedOption int, timestamp int64) error {
	// Блокируем для чтения
	qm.stateMutex.RLock()
	activeState := qm.activeQuizState
	qm.stateMutex.RUnlock()

	if activeState == nil {
		return fmt.Errorf("нет активной викторины")
	}

	return qm.answerProcessor.ProcessAnswer(
		qm.ctx, userID, questionID, selectedOption, timestamp, activeState)
}

// HandleReadyEvent обрабатывает событие готовности пользователя
func (qm *QuizManager) HandleReadyEvent(userID uint, quizID uint) error {
	return qm.answerProcessor.HandleReadyEvent(qm.ctx, userID, quizID)
}

// GetActiveQuiz возвращает активную викторину
func (qm *QuizManager) GetActiveQuiz() *entity.Quiz {
	// Блокируем для чтения
	qm.stateMutex.RLock()
	defer qm.stateMutex.RUnlock()

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
