package quizmanager

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/yourusername/trivia-api/internal/domain/entity"
)

// QuestionManager отвечает за управление вопросами, их отправку и таймеры
type QuestionManager struct {
	// Настройки
	config *Config

	// Зависимости
	deps *Dependencies

	// Канал для сигнализации о завершении вопроса
	questionDoneCh chan struct{}
}

// NewQuestionManager создает новый менеджер вопросов
func NewQuestionManager(config *Config, deps *Dependencies) *QuestionManager {
	return &QuestionManager{
		config:         config,
		deps:           deps,
		questionDoneCh: make(chan struct{}, 1),
	}
}

// QuestionDone возвращает канал для уведомления о завершении вопроса
func (qm *QuestionManager) QuestionDone() <-chan struct{} {
	return qm.questionDoneCh
}

// AutoFillQuizQuestions автоматически добавляет случайные вопросы в викторину,
// если их количество меньше установленного лимита
func (qm *QuestionManager) AutoFillQuizQuestions(ctx context.Context, quizID uint) error {
	log.Printf("[QuestionManager] Начинаю автозаполнение вопросов для викторины #%d", quizID)

	// Получаем викторину с вопросами
	quiz, err := qm.deps.QuizRepo.GetWithQuestions(quizID)
	if err != nil {
		return fmt.Errorf("не удалось получить викторину: %w", err)
	}

	// Проверяем, нужно ли добавлять вопросы
	currentCount := len(quiz.Questions)
	if currentCount >= qm.config.MaxQuestionsPerQuiz {
		log.Printf("[QuestionManager] Викторина #%d уже имеет максимальное количество вопросов (%d/%d), автозаполнение не требуется",
			quizID, currentCount, qm.config.MaxQuestionsPerQuiz)
		return nil // Уже достаточно вопросов
	}

	// Определяем, сколько вопросов нужно добавить
	neededQuestions := qm.config.MaxQuestionsPerQuiz - currentCount
	log.Printf("[QuestionManager] Викторина #%d имеет %d/%d вопросов, требуется добавить еще %d",
		quizID, currentCount, qm.config.MaxQuestionsPerQuiz, neededQuestions)

	// Создаем карту существующих ID вопросов для исключения повторов
	existingQuestionIDs := make(map[uint]bool)
	for _, q := range quiz.Questions {
		existingQuestionIDs[q.ID] = true
	}

	// Получаем случайные вопросы из базы данных
	// Запрашиваем больше вопросов, чем нужно, чтобы иметь запас для фильтрации
	randomQuestions, err := qm.deps.QuestionRepo.GetRandomQuestions(neededQuestions * 3)
	if err != nil {
		return fmt.Errorf("не удалось получить случайные вопросы: %w", err)
	}

	// Если нет доступных вопросов, возвращаем ошибку
	if len(randomQuestions) == 0 {
		return fmt.Errorf("не удалось найти вопросы для автозаполнения")
	}

	// Фильтруем вопросы, исключая те, которые уже есть в викторине
	availableQuestions := make([]entity.Question, 0)
	for _, q := range randomQuestions {
		if !existingQuestionIDs[q.ID] && q.QuizID != quizID {
			availableQuestions = append(availableQuestions, q)
		}
	}

	// Проверяем, есть ли достаточно доступных вопросов
	if len(availableQuestions) == 0 {
		return fmt.Errorf("нет доступных вопросов для автозаполнения")
	}

	// Ограничиваем количество добавляемых вопросов доступным количеством
	if neededQuestions > len(availableQuestions) {
		neededQuestions = len(availableQuestions)
		log.Printf("[QuestionManager] Доступно только %d вопросов для добавления", neededQuestions)
	}

	// Выбираем нужное количество вопросов
	selectedQuestions := availableQuestions[:neededQuestions]

	// Подготавливаем вопросы для добавления в викторину
	questionsToAdd := make([]entity.Question, len(selectedQuestions))
	for i, q := range selectedQuestions {
		// Создаем новый вопрос с корректными полями
		questionsToAdd[i] = entity.Question{
			QuizID:        quizID,
			Text:          q.Text,
			Options:       make(entity.StringArray, len(q.Options)),
			CorrectOption: q.CorrectOption,
			TimeLimitSec:  q.TimeLimitSec,
			PointValue:    q.PointValue,
		}

		// Используем встроенную функцию copy вместо цикла для копирования данных слайса
		copy(questionsToAdd[i].Options, q.Options)
	}

	// Добавляем вопросы к викторине
	if err := qm.deps.QuestionRepo.CreateBatch(questionsToAdd); err != nil {
		return fmt.Errorf("не удалось добавить вопросы: %w", err)
	}

	// Обновляем счетчик вопросов в викторине
	quiz.QuestionCount += len(questionsToAdd)
	if err := qm.deps.QuizRepo.Update(quiz); err != nil {
		return fmt.Errorf("не удалось обновить счетчик вопросов: %w", err)
	}

	log.Printf("[QuestionManager] Успешно добавлено %d вопросов в викторину #%d",
		len(questionsToAdd), quizID)

	return nil
}

// RunQuizQuestions последовательно отправляет вопросы и управляет таймерами
func (qm *QuestionManager) RunQuizQuestions(ctx context.Context, quizState *ActiveQuizState) error {
	log.Printf("[QuestionManager] Начинаю процесс отправки вопросов для викторины #%d. Всего вопросов: %d",
		quizState.Quiz.ID, len(quizState.Quiz.Questions))

	// Создаем контекст для этой конкретной викторины
	quizCtx, quizCancel := context.WithCancel(ctx)
	defer quizCancel() // Гарантируем отмену при выходе из функции

	// WaitGroup для синхронизации всех таймеров вопросов
	var timerWg sync.WaitGroup

	// Отправляем сообщение о начале викторины
	startEvent := map[string]interface{}{
		"quiz_id":        quizState.Quiz.ID,
		"title":          quizState.Quiz.Title,
		"question_count": len(quizState.Quiz.Questions),
	}

	err := qm.deps.WSManager.BroadcastEvent("quiz:start", startEvent)
	if err != nil {
		log.Printf("[QuestionManager] ОШИБКА при отправке события quiz:start для викторины #%d: %v",
			quizState.Quiz.ID, err)
		// Продолжаем, несмотря на ошибку
	}

	for i, question := range quizState.Quiz.Questions {
		// Устанавливаем текущий вопрос в состоянии
		quizState.SetCurrentQuestion(&question, i+1)

		// Добавляем задержку перед отправкой вопроса для синхронизации с фронтендом
		time.Sleep(time.Duration(qm.config.QuestionDelayMs) * time.Millisecond)

		// Получить точное время отправки вопроса
		sendTimeMs := time.Now().UnixNano() / int64(time.Millisecond)

		// Отправляем вопрос всем участникам
		questionEvent := map[string]interface{}{
			"question_id":      question.ID,
			"quiz_id":          quizState.Quiz.ID,
			"number":           i + 1,
			"text":             question.Text,
			"options":          ConvertOptionsToObjects(question.Options),
			"time_limit":       question.TimeLimitSec,
			"total_questions":  len(quizState.Quiz.Questions),
			"start_time":       sendTimeMs,
			"server_timestamp": sendTimeMs,
		}

		// Отправка с повторными попытками при ошибке
		var sendErr error
		for attempts := 0; attempts < qm.config.MaxRetries; attempts++ {
			sendErr = qm.deps.WSManager.BroadcastEvent("quiz:question", questionEvent)
			if sendErr == nil {
				log.Printf("[QuestionManager] Вопрос #%d для викторины #%d успешно отправлен с %d попытки",
					question.ID, quizState.Quiz.ID, attempts+1)
				break
			}
			log.Printf("[QuestionManager] ОШИБКА при отправке вопроса #%d для викторины #%d (попытка %d): %v",
				question.ID, quizState.Quiz.ID, attempts+1, sendErr)
			time.Sleep(qm.config.RetryInterval)
		}

		// Сохраняем время начала вопроса для подсчета времени ответа
		questionStartKey := fmt.Sprintf("question:%d:start_time", question.ID)
		if err := qm.deps.CacheRepo.Set(questionStartKey, fmt.Sprintf("%d", sendTimeMs), time.Hour); err != nil {
			log.Printf("[QuestionManager] ОШИБКА при сохранении времени начала вопроса #%d: %v", question.ID, err)
		}

		// Запускаем таймер для вопроса
		timeLimit := time.Duration(question.TimeLimitSec) * time.Second
		endTime := time.Now().Add(timeLimit)
		timerWg.Add(1)
		go qm.runQuestionTimer(quizCtx, quizState.Quiz, &question, i+1, endTime, &timerWg)

		// Ждем завершения времени на вопрос
		select {
		case <-time.After(timeLimit):
			// Продолжаем
			log.Printf("[QuestionManager] Время на вопрос #%d (%d из %d) истекло",
				question.ID, i+1, len(quizState.Quiz.Questions))
		case <-quizCtx.Done():
			log.Printf("[QuestionManager] Процесс викторины #%d был прерван на вопросе #%d",
				quizState.Quiz.ID, i+1)
			return nil
		}

		// Добавляем задержку перед отправкой правильного ответа
		time.Sleep(time.Duration(qm.config.AnswerRevealDelayMs) * time.Millisecond)

		// Отправляем правильный ответ всем участникам
		answerRevealEvent := map[string]interface{}{
			"question_id":    question.ID,
			"correct_option": question.CorrectOption,
		}

		// Отправка с повторными попытками
		for attempts := 0; attempts < qm.config.MaxRetries; attempts++ {
			sendErr = qm.deps.WSManager.BroadcastEvent("quiz:answer_reveal", answerRevealEvent)
			if sendErr == nil {
				log.Printf("[QuestionManager] Ответ на вопрос #%d успешно отправлен с %d попытки",
					question.ID, attempts+1)
				break
			}
			log.Printf("[QuestionManager] ОШИБКА при отправке ответа на вопрос #%d (попытка %d): %v",
				question.ID, attempts+1, sendErr)
			time.Sleep(qm.config.RetryInterval)
		}

		// Увеличиваем паузу между вопросами
		if i < len(quizState.Quiz.Questions)-1 {
			pauseTime := time.Duration(qm.config.InterQuestionDelayMs) * time.Millisecond
			log.Printf("[QuestionManager] Пауза %v между вопросами %d и %d",
				pauseTime, i+1, i+2)

			select {
			case <-time.After(pauseTime):
				// Продолжаем
			case <-quizCtx.Done():
				return nil
			}
		}
	}

	// Дожидаемся завершения всех таймеров перед завершением викторины
	timerWg.Wait()

	// Очищаем текущий вопрос
	quizState.ClearCurrentQuestion()

	// Отправляем сигнал о завершении всех вопросов
	select {
	case qm.questionDoneCh <- struct{}{}:
		log.Printf("[QuestionManager] Сигнал о завершении вопросов для викторины #%d отправлен", quizState.Quiz.ID)
	default:
		log.Printf("[QuestionManager] Сигнал о завершении вопросов уже был отправлен для викторины #%d", quizState.Quiz.ID)
	}

	return nil
}

// runQuestionTimer запускает таймер для вопроса и отправляет обновления
func (qm *QuestionManager) runQuestionTimer(
	ctx context.Context,
	quiz *entity.Quiz,
	question *entity.Question,
	questionNumber int,
	endTime time.Time,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	// Создаем отдельный контекст для этого таймера
	timerCtx, timerCancel := context.WithCancel(ctx)
	defer timerCancel()

	// Отправляем обновления таймера каждую секунду
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			remaining := int(time.Until(endTime).Seconds())
			if remaining <= 0 {
				// Время вышло
				log.Printf("[QuestionManager] Время на вопрос #%d (%d из %d) викторины #%d истекло",
					question.ID, questionNumber, len(quiz.Questions), quiz.ID)
				return
			}

			// Отправляем обновление таймера
			timerEvent := map[string]interface{}{
				"question_id":       question.ID,
				"remaining_seconds": remaining,
				"server_timestamp":  time.Now().UnixNano() / int64(time.Millisecond),
			}

			// Отправляем обновления таймера, но не слишком часто
			if remaining <= 5 || remaining%5 == 0 {
				if err := qm.deps.WSManager.BroadcastEvent("quiz:timer", timerEvent); err != nil {
					log.Printf("[QuestionManager] ОШИБКА при отправке таймера для вопроса #%d: %v", question.ID, err)
				} else {
					log.Printf("[QuestionManager] Таймер вопроса #%d (%d из %d): осталось %d секунд",
						question.ID, questionNumber, len(quiz.Questions), remaining)
				}
			}

		case <-timerCtx.Done():
			log.Printf("[QuestionManager] Таймер для вопроса #%d отменен", question.ID)
			return
		}
	}
}

// ConvertOptionsToObjects преобразует массив строк в массив объектов с id и text
func ConvertOptionsToObjects(options entity.StringArray) []QuestionOption {
	converted := make([]QuestionOption, len(options))

	for i, opt := range options {
		// Добавляем дополнительную проверку на пустые строки
		if opt == "" {
			opt = "(пустой вариант)"
		}

		converted[i] = QuestionOption{ID: i + 1, Text: opt}
	}

	return converted
}
