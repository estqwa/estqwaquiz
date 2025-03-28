package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

// QuizClient представляет клиента для работы с API викторины
type QuizClient struct {
	BaseURL     string
	AccessToken string
	UserID      uint
	conn        *websocket.Conn
	stopChan    chan struct{}
}

// NewQuizClient создает нового клиента для работы с API викторины
func NewQuizClient(baseURL string, accessToken string, userID uint) *QuizClient {
	return &QuizClient{
		BaseURL:     baseURL,
		AccessToken: accessToken,
		UserID:      userID,
		stopChan:    make(chan struct{}),
	}
}

// CreateQuizRequest представляет запрос на создание викторины
type CreateQuizRequest struct {
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	ScheduledTime time.Time `json:"scheduled_time"`
}

// Quiz представляет информацию о викторине
type Quiz struct {
	ID            uint      `json:"id"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	ScheduledTime time.Time `json:"scheduled_time"`
	Status        string    `json:"status"`
	QuestionCount int       `json:"question_count"`
}

// CreateQuiz создает новую викторину
func (c *QuizClient) CreateQuiz(title, description string, scheduledTime time.Time) (*Quiz, error) {
	reqBody := CreateQuizRequest{
		Title:         title,
		Description:   description,
		ScheduledTime: scheduledTime,
	}

	reqData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("ошибка маршалинга JSON: %w", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/quizzes", c.BaseURL), bytes.NewBuffer(reqData))
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.AccessToken))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var errResp map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return nil, fmt.Errorf("неожиданный статус-код: %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("ошибка API: %s", errResp["error"])
	}

	var quiz Quiz
	if err := json.NewDecoder(resp.Body).Decode(&quiz); err != nil {
		return nil, fmt.Errorf("ошибка декодирования ответа: %w", err)
	}

	return &quiz, nil
}

// Question представляет вопрос для добавления в викторину
type Question struct {
	Text          string   `json:"text"`
	Options       []string `json:"options"`
	CorrectOption int      `json:"correct_option"`
	TimeLimitSec  int      `json:"time_limit_sec"`
	PointValue    int      `json:"point_value"`
}

// AddQuestionsRequest представляет запрос на добавление вопросов
type AddQuestionsRequest struct {
	Questions []Question `json:"questions"`
}

// AddQuestions добавляет вопросы к викторине
func (c *QuizClient) AddQuestions(quizID uint, questions []Question) error {
	reqBody := AddQuestionsRequest{
		Questions: questions,
	}

	reqData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("ошибка маршалинга JSON: %w", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/quizzes/%d/questions", c.BaseURL, quizID), bytes.NewBuffer(reqData))
	if err != nil {
		return fmt.Errorf("ошибка создания запроса: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.AccessToken))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return fmt.Errorf("неожиданный статус-код: %d", resp.StatusCode)
		}
		return fmt.Errorf("ошибка API: %s", errResp["error"])
	}

	return nil
}

// ScheduleQuizRequest представляет запрос на планирование викторины
type ScheduleQuizRequest struct {
	ScheduledTime time.Time `json:"scheduled_time"`
}

// ScheduleQuiz планирует викторину на определенное время
func (c *QuizClient) ScheduleQuiz(quizID uint, scheduledTime time.Time) error {
	reqBody := ScheduleQuizRequest{
		ScheduledTime: scheduledTime,
	}

	reqData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("ошибка маршалинга JSON: %w", err)
	}

	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/api/quizzes/%d/schedule", c.BaseURL, quizID), bytes.NewBuffer(reqData))
	if err != nil {
		return fmt.Errorf("ошибка создания запроса: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.AccessToken))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return fmt.Errorf("неожиданный статус-код: %d", resp.StatusCode)
		}
		return fmt.Errorf("ошибка API: %s", errResp["error"])
	}

	return nil
}

// ConnectToQuiz подключается к викторине через WebSocket
func (c *QuizClient) ConnectToQuiz(quizID uint, onMessage func(messageType string, data map[string]interface{})) error {
	u := url.URL{
		Scheme:   "ws",
		Host:     c.BaseURL[7:], // Удаляем "http://" из начала
		Path:     "/ws",
		RawQuery: fmt.Sprintf("token=%s", c.AccessToken),
	}

	log.Printf("[BotClient] Подключение к WebSocket: %s", u.String())

	var err error
	c.conn, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("ошибка при подключении к WebSocket: %w", err)
	}

	log.Printf("[BotClient] Успешное подключение к WebSocket")

	// Отправляем сообщение о готовности
	readyMessage := map[string]interface{}{
		"type": "user:ready",
		"data": map[string]interface{}{
			"quiz_id": quizID,
		},
	}

	if err := c.conn.WriteJSON(readyMessage); err != nil {
		return fmt.Errorf("ошибка при отправке сообщения готовности: %w", err)
	}

	log.Printf("[BotClient] Отправлено сообщение готовности для викторины #%d", quizID)

	// Запускаем горутину для чтения сообщений
	go c.readMessages(onMessage)

	return nil
}

// readMessages читает сообщения из WebSocket
func (c *QuizClient) readMessages(onMessage func(messageType string, data map[string]interface{})) {
	defer c.conn.Close()

	for {
		select {
		case <-c.stopChan:
			log.Printf("[BotClient] Завершение чтения сообщений")
			return
		default:
			// Читаем сообщение
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				log.Printf("[BotClient] Ошибка при чтении сообщения: %v", err)
				return
			}

			// Разбираем JSON
			var event struct {
				Type string                 `json:"type"`
				Data map[string]interface{} `json:"data"`
			}

			if err := json.Unmarshal(message, &event); err != nil {
				log.Printf("[BotClient] Ошибка при разборе JSON: %v", err)
				continue
			}

			// Вызываем обработчик
			onMessage(event.Type, event.Data)
		}
	}
}

// SendAnswer отправляет ответ на вопрос
func (c *QuizClient) SendAnswer(questionID uint, selectedOption int) error {
	answerMessage := map[string]interface{}{
		"type": "user:answer",
		"data": map[string]interface{}{
			"question_id":     questionID,
			"selected_option": selectedOption,
			"timestamp":       time.Now().UnixNano() / int64(time.Millisecond),
		},
	}

	if err := c.conn.WriteJSON(answerMessage); err != nil {
		return fmt.Errorf("ошибка при отправке ответа: %w", err)
	}

	return nil
}

// SendAnswerWithServerTime отправляет ответ на вопрос с учетом временной синхронизации
func (c *QuizClient) SendAnswerWithServerTime(questionID uint, selectedOption int, clientTimestamp int64, serverTimestamp int64) error {
	// Вычисляем смещение времени между клиентом и сервером
	timeOffset := clientTimestamp - serverTimestamp

	// Текущее время с учетом смещения от серверного времени
	syncedTimestamp := time.Now().UnixNano()/int64(time.Millisecond) - timeOffset

	log.Printf("[BotClient] Отправка ответа с синхронизированной меткой времени: %d (смещение: %+dms)",
		syncedTimestamp, timeOffset)

	answerMessage := map[string]interface{}{
		"type": "user:answer",
		"data": map[string]interface{}{
			"question_id":     questionID,
			"selected_option": selectedOption,
			"timestamp":       syncedTimestamp,
		},
	}

	if err := c.conn.WriteJSON(answerMessage); err != nil {
		return fmt.Errorf("ошибка при отправке ответа: %w", err)
	}

	return nil
}

// SendRandomAnswer отправляет случайный ответ на вопрос (1-5)
func (c *QuizClient) SendRandomAnswer(questionID uint, options []map[string]interface{}, delay time.Duration) error {
	// Ждем указанное время
	time.Sleep(delay)

	// Выбираем случайный вариант ответа
	selectedOption := rand.Intn(len(options)) + 1

	log.Printf("[BotClient] Отправка случайного ответа %d на вопрос #%d после задержки %v",
		selectedOption, questionID, delay)

	return c.SendAnswer(questionID, selectedOption)
}

// Close закрывает соединение с сервером
func (c *QuizClient) Close() {
	close(c.stopChan)
	if c.conn != nil {
		c.conn.Close()
	}
}
