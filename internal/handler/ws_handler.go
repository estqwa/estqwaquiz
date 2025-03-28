package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	gorillaws "github.com/gorilla/websocket"
	"github.com/yourusername/trivia-api/internal/service"
	"github.com/yourusername/trivia-api/internal/websocket"
	"github.com/yourusername/trivia-api/pkg/auth"
)

// WSHandler обрабатывает WebSocket соединения
type WSHandler struct {
	wsHub       websocket.HubInterface
	wsManager   *websocket.Manager
	quizManager *service.QuizManager
	jwtService  *auth.JWTService
}

// NewWSHandler создает новый обработчик WebSocket
func NewWSHandler(
	wsHub websocket.HubInterface,
	wsManager *websocket.Manager,
	quizManager *service.QuizManager,
	jwtService *auth.JWTService,
) *WSHandler {
	handler := &WSHandler{
		wsHub:       wsHub,
		wsManager:   wsManager,
		quizManager: quizManager,
		jwtService:  jwtService,
	}

	// Регистрируем обработчики сообщений один раз при создании обработчика
	handler.registerMessageHandlers()

	return handler
}

var upgrader = gorillaws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// В продакшене здесь должна быть проверка допустимых источников
		origin := r.Header.Get("Origin")
		log.Printf("WebSocket: checking origin: %s", origin)
		// Всегда разрешаем для тестирования
		return true
	},
	// Добавляем заголовки для CORS
	EnableCompression: true,
}

// HandleConnection обрабатывает входящее WebSocket соединение
func (h *WSHandler) HandleConnection(c *gin.Context) {
	// Получаем токен из запроса
	token := c.Query("token")
	log.Printf("WebSocket: received token: %s", token)

	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing token"})
		return
	}

	// Проверяем токен
	claims, err := h.jwtService.ParseToken(token)
	if err != nil {
		log.Printf("WebSocket: Invalid token - %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	// Логируем все заголовки запроса
	log.Printf("WebSocket: Request headers:")
	for name, values := range c.Request.Header {
		for _, value := range values {
			log.Printf("  %s: %s", name, value)
		}
	}

	// Устанавливаем соединение
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Error upgrading connection: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to upgrade: %v", err)})
		return
	}

	log.Printf("WebSocket: Connection upgraded for UserID: %d", claims.UserID)

	// Создаем нового клиента
	client := websocket.NewClient(h.wsHub, conn, fmt.Sprintf("%d", claims.UserID))

	// Запускаем прослушивание сообщений
	client.StartPumps(h.wsManager.HandleMessage)
}

// registerMessageHandlers регистрирует обработчики для различных типов сообщений
func (h *WSHandler) registerMessageHandlers() {
	// Обработчик для события готовности пользователя
	h.wsManager.RegisterHandler("user:ready", func(data json.RawMessage, client *websocket.Client) error {
		var readyEvent struct {
			QuizID uint `json:"quiz_id"`
		}
		if err := json.Unmarshal(data, &readyEvent); err != nil {
			return err
		}

		userID, err := strconv.ParseUint(client.UserID, 10, 32)
		if err != nil {
			return err
		}

		return h.quizManager.HandleReadyEvent(uint(userID), readyEvent.QuizID)
	})

	// Обработчик для события ответа на вопрос
	h.wsManager.RegisterHandler("user:answer", func(data json.RawMessage, client *websocket.Client) error {
		var answerEvent struct {
			QuestionID     uint  `json:"question_id"`
			SelectedOption int   `json:"selected_option"`
			Timestamp      int64 `json:"timestamp"`
		}
		if err := json.Unmarshal(data, &answerEvent); err != nil {
			return err
		}

		userID, err := strconv.ParseUint(client.UserID, 10, 32)
		if err != nil {
			return err
		}

		return h.quizManager.ProcessAnswer(
			uint(userID),
			answerEvent.QuestionID,
			answerEvent.SelectedOption,
			answerEvent.Timestamp,
		)
	})

	// Обработчик для проверки соединения
	h.wsManager.RegisterHandler("user:heartbeat", func(data json.RawMessage, client *websocket.Client) error {
		// Отправляем ответ клиенту
		heartbeatResponse := map[string]interface{}{
			"timestamp": time.Now().UnixNano() / int64(time.Millisecond),
		}
		return h.wsManager.SendEventToUser(client.UserID, "server:heartbeat", heartbeatResponse)
	})
}
