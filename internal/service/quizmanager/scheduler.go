package quizmanager

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/yourusername/trivia-api/internal/domain/entity"
)

// Scheduler отвечает за планирование и отмену викторин
type Scheduler struct {
	// Настройки
	config *Config

	// Зависимости
	deps *Dependencies

	// Внутреннее состояние
	quizCancels sync.Map // map[uint]context.CancelFunc

	// Канал для сигнализации о запуске викторины
	quizStartCh chan uint
}

// NewScheduler создает новый планировщик викторин
func NewScheduler(config *Config, deps *Dependencies) *Scheduler {
	return &Scheduler{
		config:      config,
		deps:        deps,
		quizStartCh: make(chan uint, 10), // Буферизованный канал для событий запуска
	}
}

// GetQuizStartChannel возвращает канал для уведомлений о запуске викторин
func (s *Scheduler) GetQuizStartChannel() <-chan uint {
	return s.quizStartCh
}

// ScheduleQuiz планирует запуск викторины в заданное время
func (s *Scheduler) ScheduleQuiz(ctx context.Context, quizID uint, scheduledTime time.Time) error {
	// Сразу проверяем, что время в будущем
	if scheduledTime.Before(time.Now()) {
		return fmt.Errorf("ошибка: scheduled time is in the past")
	}

	// Получаем викторину
	quiz, err := s.deps.QuizRepo.GetWithQuestions(quizID)
	if err != nil {
		return err
	}

	// Проверяем, что у викторины есть вопросы
	if len(quiz.Questions) == 0 {
		return fmt.Errorf("quiz has no questions")
	}

	// Устанавливаем время запуска
	quiz.ScheduledTime = scheduledTime
	quiz.Status = "scheduled"

	// Сохраняем изменения
	if err := s.deps.QuizRepo.Update(quiz); err != nil {
		return err
	}

	// Создаем новый контекст для этой викторины с возможностью отмены
	quizCtx, quizCancel := context.WithCancel(ctx)

	// Сохраняем функцию отмены
	s.quizCancels.Store(quizID, quizCancel)

	// Запускаем последовательность событий в фоновом режиме
	go s.runQuizSequence(quizCtx, quiz)

	log.Printf("[Scheduler] Викторина #%d запланирована на %v", quizID, scheduledTime)
	return nil
}

// CancelQuiz отменяет запланированную викторину
func (s *Scheduler) CancelQuiz(quizID uint) error {
	// Получаем викторину
	quiz, err := s.deps.QuizRepo.GetByID(quizID)
	if err != nil {
		return err
	}

	// Проверяем, что викторина запланирована
	if !quiz.IsScheduled() {
		return fmt.Errorf("quiz is not in scheduled state")
	}

	// Получаем функцию отмены из map
	cancel, ok := s.quizCancels.Load(quizID)
	if !ok {
		log.Printf("[Scheduler] Предупреждение: функция отмены для викторины #%d не найдена", quizID)
		// Продолжаем, чтобы обновить статус в БД
	} else {
		// Вызываем функцию отмены
		cancel.(context.CancelFunc)()
		// Удаляем из map
		s.quizCancels.Delete(quizID)
		log.Printf("[Scheduler] Таймеры для викторины #%d отменены", quizID)
	}

	// Обновляем статус в БД
	if err := s.deps.QuizRepo.UpdateStatus(quizID, "cancelled"); err != nil {
		return err
	}

	// Отправляем уведомление пользователям
	cancelEvent := map[string]interface{}{
		"quiz_id": quizID,
		"message": "Quiz has been cancelled",
	}
	s.deps.WSManager.BroadcastEvent("quiz:cancelled", cancelEvent)

	log.Printf("[Scheduler] Викторина #%d отменена", quizID)
	return nil
}

// runQuizSequence выполняет последовательность событий викторины
func (s *Scheduler) runQuizSequence(ctx context.Context, quiz *entity.Quiz) {
	defer func() {
		// Удаляем функцию отмены из map при завершении последовательности
		s.quizCancels.Delete(quiz.ID)
	}()

	// Таймауты для каждого события
	autoFillTime := quiz.ScheduledTime.Add(-time.Duration(s.config.AutoFillThreshold) * time.Minute)
	announcementTime := quiz.ScheduledTime.Add(-time.Duration(s.config.AnnouncementMinutes) * time.Minute)
	waitingRoomTime := quiz.ScheduledTime.Add(-time.Duration(s.config.WaitingRoomMinutes) * time.Minute)
	countdownTime := quiz.ScheduledTime.Add(-time.Duration(s.config.CountdownSeconds) * time.Second)

	// Планируем автозаполнение вопросов, если время еще не наступило
	if autoFillTime.After(time.Now()) {
		timeToAutoFill := time.Until(autoFillTime)
		log.Printf("[Scheduler] Викторина #%d: планирую автозаполнение через %v", quiz.ID, timeToAutoFill)

		select {
		case <-time.After(timeToAutoFill):
			// Запускаем автозаполнение
			s.triggerAutoFill(ctx, quiz.ID)
		case <-ctx.Done():
			log.Printf("[Scheduler] Викторина #%d: автозаполнение отменено", quiz.ID)
			return
		}
	}

	// Планируем анонс, если время еще не наступило
	if announcementTime.After(time.Now()) {
		timeToAnnouncement := time.Until(announcementTime)
		log.Printf("[Scheduler] Викторина #%d: планирую анонс через %v", quiz.ID, timeToAnnouncement)

		select {
		case <-time.After(timeToAnnouncement):
			// Отправляем анонс
			s.triggerAnnouncement(ctx, quiz)
		case <-ctx.Done():
			log.Printf("[Scheduler] Викторина #%d: анонс отменен", quiz.ID)
			return
		}
	}

	// Планируем открытие зала ожидания, если время еще не наступило
	if waitingRoomTime.After(time.Now()) {
		timeToWaitingRoom := time.Until(waitingRoomTime)
		log.Printf("[Scheduler] Викторина #%d: планирую открытие зала ожидания через %v", quiz.ID, timeToWaitingRoom)

		select {
		case <-time.After(timeToWaitingRoom):
			// Открываем зал ожидания
			s.triggerWaitingRoom(ctx, quiz)
		case <-ctx.Done():
			log.Printf("[Scheduler] Викторина #%d: открытие зала ожидания отменено", quiz.ID)
			return
		}
	}

	// Планируем обратный отсчет, если время еще не наступило
	if countdownTime.After(time.Now()) {
		timeToCountdown := time.Until(countdownTime)
		log.Printf("[Scheduler] Викторина #%d: планирую обратный отсчет через %v", quiz.ID, timeToCountdown)

		select {
		case <-time.After(timeToCountdown):
			// Запускаем обратный отсчет
			s.triggerCountdown(ctx, quiz)
		case <-ctx.Done():
			log.Printf("[Scheduler] Викторина #%d: обратный отсчет отменен", quiz.ID)
			return
		}
	} else if time.Until(quiz.ScheduledTime) > 0 {
		// Если время для отсчета уже прошло, но викторина еще не должна начаться,
		// ждем точного времени начала
		timeToStart := time.Until(quiz.ScheduledTime)
		log.Printf("[Scheduler] Викторина #%d: слишком поздно для отсчета, ожидание начала (%v)", quiz.ID, timeToStart)

		select {
		case <-time.After(timeToStart):
			// Сигнализируем о начале викторины
			s.triggerQuizStart(ctx, quiz)
		case <-ctx.Done():
			log.Printf("[Scheduler] Викторина #%d: запуск отменен", quiz.ID)
			return
		}
	} else {
		// Если время уже прошло, сразу запускаем викторину
		log.Printf("[Scheduler] Викторина #%d: время начала уже прошло, запускаю немедленно", quiz.ID)
		s.triggerQuizStart(ctx, quiz)
	}
}

// triggerAutoFill запускает автозаполнение вопросов
func (s *Scheduler) triggerAutoFill(ctx context.Context, quizID uint) {
	log.Printf("[Scheduler] Запуск автозаполнения вопросов для викторины #%d", quizID)

	// Этот метод будет реализован в QuestionManager
	// Здесь выполняем только оповещение других компонентов
	autoFillEvent := map[string]interface{}{
		"quiz_id": quizID,
		"action":  "auto_fill",
	}
	s.deps.WSManager.BroadcastEvent("admin:quiz_action", autoFillEvent)
}

// triggerAnnouncement отправляет анонс о предстоящей викторине
func (s *Scheduler) triggerAnnouncement(ctx context.Context, quiz *entity.Quiz) {
	log.Printf("[Scheduler] Отправка анонса для викторины #%d", quiz.ID)

	// Рассчитываем оставшееся время до старта викторины
	timeToStart := time.Until(quiz.ScheduledTime)

	announcementData := map[string]interface{}{
		"quiz_id":          quiz.ID,
		"title":            quiz.Title,
		"description":      quiz.Description,
		"scheduled_time":   quiz.ScheduledTime,
		"question_count":   quiz.QuestionCount,
		"minutes_to_start": int(timeToStart.Minutes()),
	}

	s.deps.WSManager.BroadcastEvent("quiz:announcement", announcementData)
}

// triggerWaitingRoom открывает зал ожидания для викторины
func (s *Scheduler) triggerWaitingRoom(ctx context.Context, quiz *entity.Quiz) {
	log.Printf("[Scheduler] Открытие зала ожидания для викторины #%d", quiz.ID)

	// Рассчитываем оставшееся время, защищаясь от отрицательных значений
	secondsLeft := int(time.Until(quiz.ScheduledTime).Seconds())
	if secondsLeft < 0 {
		secondsLeft = 0
	}

	waitingRoomData := map[string]interface{}{
		"quiz_id":           quiz.ID,
		"title":             quiz.Title,
		"description":       quiz.Description,
		"scheduled_time":    quiz.ScheduledTime,
		"question_count":    quiz.QuestionCount,
		"starts_in_seconds": secondsLeft,
	}

	s.deps.WSManager.BroadcastEvent("quiz:waiting_room", waitingRoomData)
}

// triggerCountdown запускает обратный отсчет
func (s *Scheduler) triggerCountdown(ctx context.Context, quiz *entity.Quiz) {
	log.Printf("[Scheduler] Запуск обратного отсчета для викторины #%d", quiz.ID)

	// Создаем отдельный контекст для отсчета с возможностью отмены
	countdownCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Запускаем отсчет от N до 0 секунд
	for i := s.config.CountdownSeconds; i >= 0; i-- {
		// Проверяем контекст перед отправкой
		select {
		case <-countdownCtx.Done():
			log.Printf("[Scheduler] Обратный отсчет отменен для викторины #%d", quiz.ID)
			return
		default:
			// Продолжаем отсчет
		}

		countdownData := map[string]interface{}{
			"quiz_id":      quiz.ID,
			"seconds_left": i,
		}

		s.deps.WSManager.BroadcastEvent("quiz:countdown", countdownData)

		// Логируем только каждые 10 секунд и последние 5 секунд
		if i%10 == 0 || i < 5 {
			log.Printf("[Scheduler] Обратный отсчет викторины #%d: %d сек.", quiz.ID, i)
		}

		// Проверяем контекст перед сном, если это не последняя итерация
		if i > 0 {
			// Вычисляем точное время для следующей секунды
			nextSecond := time.Now().Add(1 * time.Second)
			timeToWait := time.Until(nextSecond)

			select {
			case <-countdownCtx.Done():
				log.Printf("[Scheduler] Обратный отсчет отменен для викторины #%d", quiz.ID)
				return
			case <-time.After(timeToWait):
				// Продолжаем отсчет
			}
		}
	}

	// Сигнализируем о начале викторины после завершения отсчета
	log.Printf("[Scheduler] Обратный отсчет завершен для викторины #%d, запуск викторины", quiz.ID)
	s.triggerQuizStart(ctx, quiz)
}

// triggerQuizStart сигнализирует о начале викторины
func (s *Scheduler) triggerQuizStart(ctx context.Context, quiz *entity.Quiz) {
	log.Printf("[Scheduler] Запуск викторины #%d", quiz.ID)

	// Обновляем статус в БД
	if err := s.deps.QuizRepo.UpdateStatus(quiz.ID, "in_progress"); err != nil {
		log.Printf("[Scheduler] Ошибка при обновлении статуса викторины #%d: %v", quiz.ID, err)
		// Продолжаем несмотря на ошибку
	}

	// Отправляем уведомление в канал для запуска викторины
	select {
	case s.quizStartCh <- quiz.ID:
		log.Printf("[Scheduler] Уведомление о запуске викторины #%d отправлено", quiz.ID)
	default:
		log.Printf("[Scheduler] Ошибка: канал запуска викторин заполнен, пропускаю уведомление для #%d", quiz.ID)
	}
}
