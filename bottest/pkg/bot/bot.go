package bot

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/yourusername/trivia-api/bottest/pkg/client"
)

// Bot представляет бота для тестирования викторины
type Bot struct {
	Name         string
	Client       *client.QuizClient
	QuizID       uint
	BotID        int
	Stats        *BotStats
	Config       *BotConfig
	IsEliminated bool
}

// BotStats хранит статистику бота
type BotStats struct {
	TotalQuestions   int
	CorrectAnswers   int
	IncorrectAnswers int
	TotalPoints      int
	AnswerTimes      []time.Duration
	Questions        []uint
	Answers          []int
	CorrectOptions   []int
	ResponseTimeMs   []int64
	ServerTimestamps []int64
	ClientTimestamps []int64
	AnswerResults    []bool
}

// BotConfig содержит настройки бота
type BotConfig struct {
	// Стратегия ответов: "random", "fast", "slow", "correct", "incorrect"
	AnswerStrategy string
	// Минимальная задержка перед ответом
	MinDelay time.Duration
	// Максимальная задержка перед ответом
	MaxDelay time.Duration
	// Процент правильных ответов (0-100), если стратегия "correct" или "incorrect"
	CorrectAnswerRate int
}

// NewBot создает нового бота
func NewBot(baseURL, token string, userID uint, botID int, config *BotConfig) *Bot {
	// Устанавливаем имя бота на основе ID
	name := fmt.Sprintf("Bot-%d", botID)

	// Инициализируем клиента
	quizClient := client.NewQuizClient(baseURL, token, userID)

	// Инициализируем статистику
	stats := &BotStats{
		AnswerTimes:      make([]time.Duration, 0),
		Questions:        make([]uint, 0),
		Answers:          make([]int, 0),
		CorrectOptions:   make([]int, 0),
		ResponseTimeMs:   make([]int64, 0),
		ServerTimestamps: make([]int64, 0),
		ClientTimestamps: make([]int64, 0),
		AnswerResults:    make([]bool, 0),
	}

	return &Bot{
		Name:   name,
		Client: quizClient,
		BotID:  botID,
		Stats:  stats,
		Config: config,
	}
}

// CreateAndJoinQuiz создает викторину и присоединяется к ней
func (b *Bot) CreateAndJoinQuiz() error {
	// Создаем викторину, которая начнется через 1 минуту
	startTime := time.Now().Add(1 * time.Minute)

	log.Printf("[%s] Создание викторины, запланированной на %v", b.Name, startTime)

	quiz, err := b.Client.CreateQuiz(
		fmt.Sprintf("Тестовая викторина от %s", b.Name),
		"Автоматически созданная викторина для тестирования",
		startTime,
	)
	if err != nil {
		return fmt.Errorf("ошибка при создании викторины: %w", err)
	}

	b.QuizID = quiz.ID
	log.Printf("[%s] Викторина #%d создана, начало в %v", b.Name, b.QuizID, quiz.ScheduledTime)

	// Добавляем вопросы к викторине
	err = b.addTestQuestions(quiz.ID)
	if err != nil {
		return fmt.Errorf("ошибка при добавлении вопросов: %w", err)
	}

	// Подтверждаем время начала викторины
	err = b.Client.ScheduleQuiz(quiz.ID, startTime)
	if err != nil {
		return fmt.Errorf("ошибка при планировании викторины: %w", err)
	}

	log.Printf("[%s] Викторина #%d запланирована на %v", b.Name, b.QuizID, startTime)

	// Подключаемся к викторине
	return b.JoinQuiz(quiz.ID)
}

// JoinQuiz присоединяется к существующей викторине
func (b *Bot) JoinQuiz(quizID uint) error {
	b.QuizID = quizID
	log.Printf("[%s] Подключение к викторине #%d", b.Name, quizID)

	// Подключаемся к викторине через WebSocket
	return b.Client.ConnectToQuiz(quizID, b.handleMessage)
}

// handleMessage обрабатывает входящие сообщения
func (b *Bot) handleMessage(messageType string, data map[string]interface{}) {
	log.Printf("[%s] Получено сообщение типа: %s", b.Name, messageType)

	switch messageType {
	case "quiz:start":
		b.handleQuizStart(data)
	case "quiz:countdown":
		b.handleQuizCountdown(data)
	case "quiz:question":
		b.handleQuizQuestion(data)
	case "quiz:timer":
		b.handleQuizTimer(data)
	case "quiz:answer_result":
		b.handleAnswerResult(data)
	case "quiz:elimination":
		b.handleElimination(data)
	case "quiz:elimination_reminder":
		log.Printf("[%s] Напоминание о выбывании получено", b.Name)
	case "quiz:leaderboard":
		b.handleLeaderboard(data)
	case "quiz:end":
		b.handleQuizEnd(data)
	case "server:heartbeat":
		// Ничего не делаем, просто логируем
		log.Printf("[%s] Heartbeat получен: %v", b.Name, data["timestamp"])
	default:
		log.Printf("[%s] Необработанный тип сообщения: %s", b.Name, messageType)
	}
}

// handleQuizStart обрабатывает начало викторины
func (b *Bot) handleQuizStart(data map[string]interface{}) {
	log.Printf("[%s] Викторина началась! ID: %v, Название: %v, Вопросов: %v",
		b.Name, data["quiz_id"], data["title"], data["question_count"])
}

// handleQuizCountdown обрабатывает обратный отсчет
func (b *Bot) handleQuizCountdown(data map[string]interface{}) {
	secondsLeft, ok := data["seconds_left"].(float64)
	if !ok {
		log.Printf("[%s] Ошибка: не удалось получить seconds_left", b.Name)
		return
	}

	if int(secondsLeft) <= 5 || int(secondsLeft)%10 == 0 {
		log.Printf("[%s] Обратный отсчет: %d секунд", b.Name, int(secondsLeft))
	}
}

// handleQuizQuestion обрабатывает вопрос
func (b *Bot) handleQuizQuestion(data map[string]interface{}) {
	// Получаем ID вопроса
	questionID, ok := data["question_id"].(float64)
	if !ok {
		log.Printf("[%s] Ошибка: не удалось получить question_id", b.Name)
		return
	}

	// Получаем текст вопроса
	questionText, ok := data["text"].(string)
	if !ok {
		log.Printf("[%s] Ошибка: не удалось получить текст вопроса", b.Name)
		return
	}

	// Сокращаем длинный текст вопроса для логов
	if len(questionText) > 50 {
		questionText = questionText[:50] + "..."
	}

	// Получаем варианты ответов
	optionsRaw, ok := data["options"].([]interface{})
	if !ok {
		log.Printf("[%s] Ошибка: не удалось получить варианты ответов", b.Name)
		return
	}

	// Конвертируем в правильный формат
	options := make([]map[string]interface{}, len(optionsRaw))
	for i, opt := range optionsRaw {
		options[i], ok = opt.(map[string]interface{})
		if !ok {
			log.Printf("[%s] Ошибка: неверный формат варианта ответа", b.Name)
			return
		}
	}

	// Получаем номер вопроса
	number, ok := data["number"].(float64)
	if !ok {
		log.Printf("[%s] Ошибка: не удалось получить номер вопроса", b.Name)
		return
	}

	// Получаем общее количество вопросов
	totalQuestions, ok := data["total_questions"].(float64)
	if !ok {
		log.Printf("[%s] Ошибка: не удалось получить общее количество вопросов", b.Name)
		return
	}

	// Получаем ограничение времени
	timeLimit, ok := data["time_limit"].(float64)
	if !ok {
		log.Printf("[%s] Ошибка: не удалось получить ограничение времени", b.Name)
		return
	}

	// Получаем серверную метку времени
	serverTimestamp, ok := data["server_timestamp"].(float64)
	if !ok {
		log.Printf("[%s] Ошибка: не удалось получить серверную метку времени", b.Name)
		return
	}

	// Текущее время клиента
	clientReceivedAt := time.Now().UnixNano() / int64(time.Millisecond)

	// Вычисляем разницу между временем сервера и клиента
	timeOffset := clientReceivedAt - int64(serverTimestamp)

	// Логируем информацию о синхронизации времени
	log.Printf(`[%s] {
  "event": "QUIZ_QUESTION",
  "question_id": %d,
  "server_timestamp": %d,
  "client_received_at": %d,
  "offset_ms": "%+dms"
}`, b.Name, uint(questionID), int64(serverTimestamp), clientReceivedAt, timeOffset)

	log.Printf("[%s] Получен вопрос #%d/%d: %s (ID: %d, время: %.0f сек)",
		b.Name, int(number), int(totalQuestions), questionText, uint(questionID), timeLimit)

	// Сохраняем серверную метку времени и время клиента
	b.Stats.ServerTimestamps = append(b.Stats.ServerTimestamps, int64(serverTimestamp))
	b.Stats.Questions = append(b.Stats.Questions, uint(questionID))

	// Если бот не выбыл, отправляем ответ
	if !b.IsEliminated {
		go b.sendRandomAnswerWithStrategy(uint(questionID), options, int64(serverTimestamp), timeLimit)
	} else {
		log.Printf("[%s] Бот выбыл, не отправляет ответ на вопрос #%d", b.Name, uint(questionID))
	}
}

// handleQuizTimer обрабатывает сообщение о таймере
func (b *Bot) handleQuizTimer(data map[string]interface{}) {
	questionID, ok := data["question_id"].(float64)
	if !ok {
		log.Printf("[%s] Ошибка: не удалось получить question_id в таймере", b.Name)
		return
	}

	secondsLeft, ok := data["seconds_left"].(float64)
	if !ok {
		log.Printf("[%s] Ошибка: не удалось получить seconds_left в таймере", b.Name)
		return
	}

	// Логируем только определенные значения, чтобы не засорять вывод
	if secondsLeft <= 5 || int(secondsLeft)%5 == 0 {
		log.Printf("[%s] Таймер вопроса #%d: осталось %.0f секунд",
			b.Name, uint(questionID), secondsLeft)
	}
}

// handleAnswerResult обрабатывает результат ответа
func (b *Bot) handleAnswerResult(data map[string]interface{}) {
	questionID, ok := data["question_id"].(float64)
	if !ok {
		log.Printf("[%s] Ошибка: не удалось получить question_id в результате", b.Name)
		return
	}

	isCorrect, ok := data["is_correct"].(bool)
	if !ok {
		log.Printf("[%s] Ошибка: не удалось получить is_correct в результате", b.Name)
		return
	}

	points, ok := data["points_earned"].(float64)
	if !ok {
		log.Printf("[%s] Ошибка: не удалось получить points_earned в результате", b.Name)
		return
	}

	timeTaken, ok := data["time_taken_ms"].(float64)
	if !ok {
		log.Printf("[%s] Ошибка: не удалось получить time_taken_ms в результате", b.Name)
		return
	}

	correctOption, ok := data["correct_option"].(float64)
	if !ok {
		log.Printf("[%s] Ошибка: не удалось получить correct_option в результате", b.Name)
		return
	}

	yourAnswer, ok := data["your_answer"].(float64)
	if !ok {
		log.Printf("[%s] Ошибка: не удалось получить your_answer в результате", b.Name)
		return
	}

	isEliminated, ok := data["is_eliminated"].(bool)
	if ok && isEliminated {
		b.IsEliminated = true
		log.Printf("[%s] ВЫ ВЫБЫЛИ из викторины после ответа на вопрос #%d",
			b.Name, uint(questionID))
	}

	// Обновляем статистику
	b.Stats.AnswerResults = append(b.Stats.AnswerResults, isCorrect)
	b.Stats.ResponseTimeMs = append(b.Stats.ResponseTimeMs, int64(timeTaken))
	b.Stats.CorrectOptions = append(b.Stats.CorrectOptions, int(correctOption))
	b.Stats.Answers = append(b.Stats.Answers, int(yourAnswer))

	if isCorrect {
		b.Stats.CorrectAnswers++
	} else {
		b.Stats.IncorrectAnswers++
	}

	b.Stats.TotalPoints += int(points)
	b.Stats.TotalQuestions++

	log.Printf("[%s] Результат ответа на вопрос #%d: %s, время: %.0f мс, получено очков: %.0f, ваш ответ: %.0f, правильный: %.0f",
		b.Name, uint(questionID),
		func() string {
			if isCorrect {
				return "ВЕРНО"
			}
			return "НЕВЕРНО"
		}(),
		timeTaken, points, yourAnswer, correctOption)
}

// handleElimination обрабатывает сообщение о выбывании
func (b *Bot) handleElimination(data map[string]interface{}) {
	b.IsEliminated = true

	message, ok := data["message"].(string)
	if !ok {
		log.Printf("[%s] Ошибка: не удалось получить message в сообщении о выбывании", b.Name)
		return
	}

	reason, ok := data["reason"].(string)
	if !ok {
		log.Printf("[%s] Ошибка: не удалось получить reason в сообщении о выбывании", b.Name)
		return
	}

	log.Printf("[%s] ВЫ ВЫБЫЛИ из викторины! Сообщение: %s, Причина: %s",
		b.Name, message, reason)
}

// handleLeaderboard обрабатывает таблицу лидеров
func (b *Bot) handleLeaderboard(data map[string]interface{}) {
	log.Printf("[%s] Получена таблица лидеров", b.Name)
}

// handleQuizEnd обрабатывает завершение викторины
func (b *Bot) handleQuizEnd(data map[string]interface{}) {
	log.Printf("[%s] Викторина завершена! Итоговая статистика:", b.Name)
	log.Printf("[%s] Всего вопросов: %d", b.Name, b.Stats.TotalQuestions)
	log.Printf("[%s] Правильных ответов: %d", b.Name, b.Stats.CorrectAnswers)
	log.Printf("[%s] Неправильных ответов: %d", b.Name, b.Stats.IncorrectAnswers)
	log.Printf("[%s] Всего очков: %d", b.Name, b.Stats.TotalPoints)
}

// sendRandomAnswerWithStrategy отправляет ответ по выбранной стратегии
func (b *Bot) sendRandomAnswerWithStrategy(questionID uint, options []map[string]interface{}, serverTimestamp int64, timeLimit float64) {
	var delay time.Duration
	var selectedOption int

	switch b.Config.AnswerStrategy {
	case "fast":
		// Быстрый ответ, минимальная задержка
		delay = b.Config.MinDelay
	case "slow":
		// Медленный ответ, максимальная задержка
		delay = b.Config.MaxDelay
	case "correct":
		// Вероятность правильного ответа по заданному проценту
		// Но здесь нам неизвестен правильный ответ до получения результата
		// Поэтому мы просто случайно выбираем ответ
		delay = b.randomDelay()
	case "incorrect":
		// Вероятность неправильного ответа по заданному проценту
		// Аналогично предыдущему, просто выбираем случайно
		delay = b.randomDelay()
	default: // "random"
		// Случайная задержка между минимальной и максимальной
		delay = b.randomDelay()
	}

	// Проверяем, не слишком ли большая задержка (критический порог - 10 секунд)
	maxAllowedDelay := time.Duration((timeLimit*1000)-500) * time.Millisecond // -500мс запас
	criticalThreshold := time.Duration(10000) * time.Millisecond              // 10 секунд - порог выбывания

	if delay > maxAllowedDelay {
		log.Printf("[%s] ⚠️ Внимание: задержка %v превышает лимит времени вопроса. Корректируем до %v",
			b.Name, delay, maxAllowedDelay)
		delay = maxAllowedDelay
	}

	if delay > criticalThreshold {
		log.Printf("[%s] ⚠️ Внимание: задержка %v превышает критический порог выбывания (10 сек). Корректируем до 9.5 сек",
			b.Name, delay)
		delay = time.Duration(9500) * time.Millisecond // 9.5 секунд
	}

	// Выбираем случайный вариант ответа
	selectedOption = rand.Intn(len(options)) + 1

	// Расчетное время для отправки ответа (время сервера + задержка)
	calculatedAnswerTime := serverTimestamp + int64(delay/time.Millisecond)

	// Ждем указанное время с учетом времени сервера
	time.Sleep(delay)

	// Фиксируем время отправки ответа
	clientTimestamp := time.Now().UnixNano() / int64(time.Millisecond)

	// Вычисляем разницу между расчетным и фактическим временем отправки
	timingDiff := clientTimestamp - calculatedAnswerTime

	// Логируем точную информацию о синхронизации ответа
	log.Printf(`[%s] {
  "event": "QUIZ_ANSWER_SENT",
  "question_id": %d,
  "server_timestamp": %d,
  "calculated_answer_time": %d,
  "actual_sent_time": %d,
  "timing_diff": "%+dms"
}`, b.Name, questionID, serverTimestamp, calculatedAnswerTime, clientTimestamp, timingDiff)

	log.Printf("[%s] Отправка ответа %d на вопрос #%d после задержки %v",
		b.Name, selectedOption, questionID, delay)

	b.Stats.ClientTimestamps = append(b.Stats.ClientTimestamps, clientTimestamp)

	// Отправляем ответ с синхронизированным временем сервера
	if err := b.Client.SendAnswerWithServerTime(questionID, selectedOption, clientTimestamp, serverTimestamp); err != nil {
		log.Printf("[%s] Ошибка при отправке ответа: %v", b.Name, err)
	}
}

// randomDelay возвращает случайную задержку между минимальной и максимальной
func (b *Bot) randomDelay() time.Duration {
	delayRange := b.Config.MaxDelay - b.Config.MinDelay
	if delayRange <= 0 {
		return b.Config.MinDelay
	}

	randomMs := rand.Int63n(int64(delayRange))
	return b.Config.MinDelay + time.Duration(randomMs)
}

// addTestQuestions добавляет тестовые вопросы к викторине
func (b *Bot) addTestQuestions(quizID uint) error {
	log.Printf("[%s] Добавление тестовых вопросов к викторине #%d", b.Name, quizID)

	questions := []client.Question{
		{
			Text:          "Какой язык программирования был создан в Google для замены C++?",
			Options:       []string{"Java", "Go", "Rust", "Swift", "Kotlin"},
			CorrectOption: 1, // Go
			TimeLimitSec:  15,
			PointValue:    10,
		},
		{
			Text:          "Какая структура данных работает по принципу LIFO?",
			Options:       []string{"Очередь", "Стек", "Список", "Дерево", "Граф"},
			CorrectOption: 1, // Стек
			TimeLimitSec:  10,
			PointValue:    15,
		},
		{
			Text:          "Что такое горутины в Go?",
			Options:       []string{"Функции", "Легковесные потоки", "Каналы", "Структуры", "Интерфейсы"},
			CorrectOption: 1, // Легковесные потоки
			TimeLimitSec:  20,
			PointValue:    20,
		},
		{
			Text:          "Какой протокол используется для загрузки веб-страниц?",
			Options:       []string{"FTP", "SMTP", "HTTP", "SSH", "DNS"},
			CorrectOption: 2, // HTTP
			TimeLimitSec:  10,
			PointValue:    5,
		},
		{
			Text:          "Что такое WebSocket?",
			Options:       []string{"Протокол для односторонней связи", "Библиотека JavaScript", "Протокол для двусторонней связи", "Веб-сервер", "Фреймворк"},
			CorrectOption: 2, // Протокол для двусторонней связи
			TimeLimitSec:  15,
			PointValue:    15,
		},
	}

	return b.Client.AddQuestions(quizID, questions)
}
