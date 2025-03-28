package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/yourusername/trivia-api/bottest/pkg/bot"
)

var (
	// Адрес сервера по умолчанию
	baseURL string = "http://localhost:8080"
	// JWT токен для авторизации
	token string
	// ID викторины (для подключения к существующей)
	quizID uint
	// Количество ботов
	botCount int = 1
	// Стратегия ответов
	answerStrategy string = "random"
	// Минимальная задержка перед ответом (в миллисекундах)
	minDelayMs int = 1000
	// Максимальная задержка перед ответом (в миллисекундах)
	maxDelayMs int = 5000
	// Процент правильных ответов (для стратегий correct/incorrect)
	correctAnswerRate int = 50
	// ID пользователя (начальное значение)
	startUserID uint = 1000
	// Создать новую викторину
	createQuiz bool
)

func main() {
	// Корневая команда
	rootCmd := &cobra.Command{
		Use:   "bottest",
		Short: "Тестовый клиент для викторины",
		Long:  "Тестовый клиент для проверки функциональности API викторины.",
	}

	// Команда запуска ботов
	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Запустить ботов для тестирования викторины",
		Run:   runBots,
	}

	// Добавляем флаги для настройки
	runCmd.Flags().StringVar(&baseURL, "url", baseURL, "Базовый URL API (например, http://localhost:8080)")
	runCmd.Flags().StringVar(&token, "token", "", "JWT токен для авторизации")
	runCmd.Flags().UintVar(&quizID, "quiz", 0, "ID существующей викторины для подключения")
	runCmd.Flags().IntVar(&botCount, "bots", botCount, "Количество ботов для запуска")
	runCmd.Flags().StringVar(&answerStrategy, "strategy", answerStrategy, "Стратегия ответов: random, fast, slow, correct, incorrect")
	runCmd.Flags().IntVar(&minDelayMs, "min-delay", minDelayMs, "Минимальная задержка перед ответом (мс)")
	runCmd.Flags().IntVar(&maxDelayMs, "max-delay", maxDelayMs, "Максимальная задержка перед ответом (мс)")
	runCmd.Flags().IntVar(&correctAnswerRate, "correct-rate", correctAnswerRate, "Процент правильных ответов (0-100)")
	runCmd.Flags().UintVar(&startUserID, "start-uid", startUserID, "Начальный ID пользователя")
	runCmd.Flags().BoolVar(&createQuiz, "create", false, "Создать новую викторину")

	// Проверка обязательных параметров
	runCmd.MarkFlagRequired("token")

	// Добавляем подкоманды к корневой команде
	rootCmd.AddCommand(runCmd)

	// Запускаем
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// runBots запускает указанное количество ботов
func runBots(cmd *cobra.Command, args []string) {
	// Проверки входных параметров
	if token == "" {
		log.Fatal("Требуется JWT токен (--token)")
	}

	if quizID == 0 && !createQuiz {
		log.Fatal("Требуется указать ID викторины (--quiz) или создать новую (--create)")
	}

	if !isValidStrategy(answerStrategy) {
		log.Fatal("Неверная стратегия ответов. Допустимые значения: random, fast, slow, correct, incorrect")
	}

	if minDelayMs < 0 {
		log.Fatal("Минимальная задержка не может быть отрицательной")
	}

	if maxDelayMs < minDelayMs {
		log.Fatal("Максимальная задержка должна быть больше или равна минимальной")
	}

	if correctAnswerRate < 0 || correctAnswerRate > 100 {
		log.Fatal("Процент правильных ответов должен быть в диапазоне 0-100")
	}

	// Инициализируем rand с текущим временем
	rand.Seed(time.Now().UnixNano())

	log.Printf("🚀 Запуск тестового клиента для викторины")
	log.Printf("📡 Сервер: %s", baseURL)
	log.Printf("🤖 Ботов: %d", botCount)
	log.Printf("⚙️ Стратегия: %s", answerStrategy)
	log.Printf("⏱️ Задержка: %d-%d мс", minDelayMs, maxDelayMs)

	// Если нужно создать викторину, делаем это
	var createdQuizID uint
	if createQuiz {
		log.Printf("🛠️ Создание новой викторины...")
		var err error
		createdQuizID, err = createNewQuiz()
		if err != nil {
			log.Fatalf("❌ Ошибка при создании викторины: %v", err)
		}
		quizID = createdQuizID
		log.Printf("✅ Викторина #%d создана успешно! Ожидаем начала...", quizID)
	}

	// Создаем и запускаем ботов
	var wg sync.WaitGroup
	bots := make([]*bot.Bot, botCount)

	for i := 0; i < botCount; i++ {
		// Создаем конфигурацию бота
		config := &bot.BotConfig{
			AnswerStrategy:    answerStrategy,
			MinDelay:          time.Duration(minDelayMs) * time.Millisecond,
			MaxDelay:          time.Duration(maxDelayMs) * time.Millisecond,
			CorrectAnswerRate: correctAnswerRate,
		}

		// Каждому боту назначаем свой userID
		userID := startUserID + uint(i)

		// Создаем бота
		b := bot.NewBot(baseURL, token, userID, i+1, config)
		bots[i] = b

		// Запускаем бота в отдельной горутине
		wg.Add(1)
		go func(b *bot.Bot) {
			defer wg.Done()
			var err error
			if createQuiz && i == 0 && createdQuizID > 0 {
				// Первый бот уже создал викторину, остальные подключаются
				log.Printf("[%s] Подключение к викторине #%d, созданной первым ботом", b.Name, createdQuizID)
				err = b.JoinQuiz(createdQuizID)
			} else if createQuiz && i == 0 {
				// Первый бот создает викторину
				log.Printf("[%s] Создание новой викторины и подключение", b.Name)
				err = b.CreateAndJoinQuiz()
			} else {
				// Подключаемся к указанной викторине
				log.Printf("[%s] Подключение к викторине #%d", b.Name, quizID)
				err = b.JoinQuiz(quizID)
			}

			if err != nil {
				log.Printf("[%s] ❌ Ошибка: %v", b.Name, err)
			}
		}(b)
	}

	// Обработка сигналов для корректного завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("🛑 Получен сигнал завершения, закрываем соединения...")
		for _, b := range bots {
			if b != nil && b.Client != nil {
				b.Client.Close()
			}
		}
		// Ждем некоторое время и выходим
		time.Sleep(1 * time.Second)
		os.Exit(0)
	}()

	log.Printf("📊 Боты запущены! Нажмите Ctrl+C для завершения...")
	wg.Wait()
}

// createNewQuiz создает новую викторину и возвращает ее ID
func createNewQuiz() (uint, error) {
	// Создаем конфигурацию для бота-создателя
	config := &bot.BotConfig{
		AnswerStrategy:    "random",
		MinDelay:          time.Duration(minDelayMs) * time.Millisecond,
		MaxDelay:          time.Duration(maxDelayMs) * time.Millisecond,
		CorrectAnswerRate: correctAnswerRate,
	}

	// Создаем бота с ID 999 для создания викторины
	creatorBot := bot.NewBot(baseURL, token, startUserID, 999, config)

	// Создаем викторину, запланированную на ближайшее время
	startTime := time.Now().Add(1 * time.Minute)

	// Создаем викторину
	quiz, err := creatorBot.Client.CreateQuiz(
		"Тестовая викторина",
		"Автоматически созданная викторина для тестирования ботов",
		startTime,
	)
	if err != nil {
		return 0, fmt.Errorf("ошибка при создании викторины: %w", err)
	}

	// Добавляем вопросы
	questions := []struct {
		Text          string
		Options       []string
		CorrectOption int
		TimeLimitSec  int
		PointValue    int
	}{
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

	// Преобразуем вопросы в нужный формат
	apiQuestions := make([]struct {
		Text          string   `json:"text"`
		Options       []string `json:"options"`
		CorrectOption int      `json:"correct_option"`
		TimeLimitSec  int      `json:"time_limit_sec"`
		PointValue    int      `json:"point_value"`
	}, len(questions))

	for i, q := range questions {
		apiQuestions[i].Text = q.Text
		apiQuestions[i].Options = q.Options
		apiQuestions[i].CorrectOption = q.CorrectOption
		apiQuestions[i].TimeLimitSec = q.TimeLimitSec
		apiQuestions[i].PointValue = q.PointValue
	}

	// Закрываем соединение бота-создателя
	creatorBot.Client.Close()

	return quiz.ID, nil
}

// isValidStrategy проверяет, является ли стратегия допустимой
func isValidStrategy(strategy string) bool {
	validStrategies := []string{"random", "fast", "slow", "correct", "incorrect"}
	strategy = strings.ToLower(strategy)

	for _, valid := range validStrategies {
		if strategy == valid {
			return true
		}
	}

	return false
}
