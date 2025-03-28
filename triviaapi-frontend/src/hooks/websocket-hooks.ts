import { useEffect, useCallback, useState } from 'react';
import { useAppDispatch, useAppSelector } from './redux-hooks';
import { 
  setActiveQuiz,
  setCurrentQuestion,
  updateUserAnswer,
  updateRemainingTime,
  endCurrentQuestion,
  setQuizEnded,
  setUserResults,
  setLeaderboard,
  setQuizCancelled
} from '../store/quiz/slice';
import { 
  WebSocketEventType,
  QuizStartEvent,
  QuestionStartEvent,
  QuestionEndEvent,
  ResultUpdateEvent,
  QuizEndEvent,
  QuizTimerEvent,
  QuizCancelledEvent,
  WebSocketMessage,
  UserAnswerEvent,
  QuizAnnouncementEvent,
  QuizWaitingRoomEvent,
  QuizCountdownEvent,
  QuizAnswerRevealEvent,
  QuizAnswerResultEvent,
  QuizLeaderboardEvent
} from '../types/websocket';
import { apiClient } from '../api/http/client'; // Импортируем HTTP клиент для вызова API

// URL для WebSocket соединения
const WS_URL = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8080/ws';

// Класс для управления WebSocket подключением
class WebSocketManager {
  private static instance: WebSocketManager;
  private socket: WebSocket | null = null;
  private messageHandlers: Record<string, Set<(data: unknown) => void>> = {};
  private reconnectTimeout: ReturnType<typeof setTimeout> | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000; // Начальная задержка для переподключения (1 секунда)
  private connected = false;
  private connecting = false;

  // Приватный конструктор для Singleton паттерна
  private constructor() {}

  // Получение экземпляра класса
  public static getInstance(): WebSocketManager {
    if (!WebSocketManager.instance) {
      WebSocketManager.instance = new WebSocketManager();
    }
    return WebSocketManager.instance;
  }

  // Подключение к WebSocket серверу
  public connect(token: string): Promise<void> {
    if (this.connected) {
      return Promise.resolve();
    }

    if (this.connecting) {
      return new Promise((resolve, reject) => {
        const checkInterval = setInterval(() => {
          if (this.connected) {
            clearInterval(checkInterval);
            resolve();
          } else if (!this.connecting) {
            clearInterval(checkInterval);
            reject(new Error('Connection failed'));
          }
        }, 100);
      });
    }

    this.connecting = true;
    
    return new Promise((resolve, reject) => {
      try {
        this.socket = new WebSocket(`${WS_URL}?token=${token}`);
        
        this.socket.onopen = () => {
          console.log('WebSocket connected');
          this.connected = true;
          this.connecting = false;
          this.reconnectAttempts = 0;
          this.reconnectDelay = 1000;
          resolve();
        };
        
        this.socket.onclose = (event) => {
          console.log('WebSocket closed', event.code, event.reason);
          this.connected = false;
          this.connecting = false;
          
          // Если соединение закрыто не специально (код 1000), пытаемся переподключиться
          if (event.code !== 1000) {
            this.attemptReconnect(token);
          }
        };
        
        this.socket.onerror = (error) => {
          console.error('WebSocket error:', error);
          this.connecting = false;
          if (!this.connected) {
            reject(error);
          }
        };
        
        this.socket.onmessage = (event) => {
          try {
            const message = JSON.parse(event.data) as WebSocketMessage;
            console.log('Received WebSocket message:', message);
            
            // Проверяем наличие обработчиков для типа сообщения
            const handlers = this.messageHandlers[message.type];
            if (handlers) {
              handlers.forEach(handler => {
                try {
                  handler(message.data);
                } catch (handlerError) {
                  console.error(`Error in WebSocket handler for ${message.type}:`, handlerError);
                }
              });
            }
          } catch (error) {
            console.error('Error parsing WebSocket message:', error);
          }
        };
      } catch (error) {
        console.error('Error creating WebSocket connection:', error);
        this.connecting = false;
        reject(error);
      }
    });
  }
  
  // Отправка сообщения через WebSocket
  public sendMessage<T>(type: string, data: T): boolean {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
      console.error(`Cannot send message, WebSocket not connected, type: ${type}`);
      return false;
    }
    
    try {
      const message: WebSocketMessage = { type, data };
      this.socket.send(JSON.stringify(message));
      return true;
    } catch (error) {
      console.error('Error sending WebSocket message:', error);
      return false;
    }
  }
  
  // Закрытие WebSocket соединения
  public disconnect(): void {
    if (this.socket) {
      this.socket.close(1000, 'Disconnected by client');
      this.socket = null;
    }
    
    // Очищаем таймер переподключения
    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout);
      this.reconnectTimeout = null;
    }
    
    this.connected = false;
    this.connecting = false;
    console.log('WebSocket disconnected');
  }
  
  // Попытка переподключения после потери соединения
  private attemptReconnect(token: string): void {
    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout);
    }
    
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('Max WebSocket reconnect attempts reached');
      return;
    }
    
    this.reconnectAttempts++;
    const delay = Math.min(30000, this.reconnectDelay * Math.pow(1.5, this.reconnectAttempts - 1));
    
    console.log(`Attempting to reconnect WebSocket in ${delay}ms (attempt ${this.reconnectAttempts})`);
    
    this.reconnectTimeout = setTimeout(() => {
      this.connect(token)
        .then(() => {
          console.log('WebSocket reconnected successfully');
        })
        .catch(error => {
          console.error('WebSocket reconnect failed:', error);
        });
    }, delay);
  }
  
  // Добавление обработчика для определенного типа сообщений
  public addMessageHandler<T>(type: string, handler: (data: T) => void): () => void {
    if (!this.messageHandlers[type]) {
      this.messageHandlers[type] = new Set();
    }
    
    this.messageHandlers[type].add(handler as (data: unknown) => void);
    console.log(`Added WebSocket message handler for type: ${type}`);
    
    // Возвращаем функцию для удаления этого обработчика
    return () => {
      if (this.messageHandlers[type]) {
        this.messageHandlers[type].delete(handler as (data: unknown) => void);
        if (this.messageHandlers[type].size === 0) {
          delete this.messageHandlers[type];
        }
      }
    };
  }
  
  // Получение статуса соединения
  public isConnected(): boolean {
    return this.connected;
  }
}

// Инициализация WebSocket клиента
export const wsClient = WebSocketManager.getInstance();

/**
 * Хук для подключения к WebSocket и обработки сообщений для активных викторин
 */
export const useQuizWebSocket = (quizId?: number) => {
  const dispatch = useAppDispatch();
  const { token } = useAppSelector(state => state.auth);
  const [isConnected, setIsConnected] = useState(wsClient.isConnected());
  
  // Подключение к WebSocket
  useEffect(() => {
    if (!token) return;
    
    let connectionCheck: ReturnType<typeof setInterval>;
    
    const connect = async () => {
      try {
        await wsClient.connect(token);
        setIsConnected(true);
      } catch (error) {
        console.error('Failed to connect to WebSocket:', error);
        setIsConnected(false);
      }
    };
    
    // Инициируем подключение
    connect();
    
    // Периодически проверяем состояние соединения
    connectionCheck = setInterval(() => {
      setIsConnected(wsClient.isConnected());
    }, 5000);
    
    return () => {
      clearInterval(connectionCheck);
    };
  }, [token]);
  
  // Регистрация обработчиков сообщений, связанных с викторинами
  useEffect(() => {
    if (!quizId) return;
    
    const handlers: (() => void)[] = [];
    
    // Обработчик начала викторины
    handlers.push(wsClient.addMessageHandler<QuizStartEvent>(WebSocketEventType.QUIZ_START, (data: QuizStartEvent) => {
      // Если прислали другую викторину, игнорируем
      if (data.quizId !== quizId) return;
      
      dispatch(setActiveQuiz({
        quiz: {
          id: data.quizId,
          title: data.title,
          description: data.description || '',
          questionCount: data.numQuestions,
          status: 'active',
          startTime: data.startTime,
          durationMinutes: data.durationMinutes
        }
      }));
    }));
    
    // Обработчик событий анонса викторины
    handlers.push(wsClient.addMessageHandler<QuizAnnouncementEvent>(WebSocketEventType.QUIZ_ANNOUNCEMENT, (data) => {
      // Обработка анонса викторины
      console.log('Объявлена новая викторина:', data);
      // Мы не обновляем Redux тут, нужно получить данные через API для полного объекта
    }));

    // Обработчик открытия зала ожидания
    handlers.push(wsClient.addMessageHandler<QuizWaitingRoomEvent>(WebSocketEventType.QUIZ_WAITING_ROOM, (data) => {
      // Обработка события открытия зала ожидания
      console.log('Открыт зал ожидания викторины:', data);
      // Можно обновить счетчик обратного отсчета в UI
    }));

    // Обработчик обратного отсчета
    handlers.push(wsClient.addMessageHandler<QuizCountdownEvent>(WebSocketEventType.QUIZ_COUNTDOWN, (data) => {
      // Если прислали данные для другой викторины, игнорируем
      if (data.quizId !== quizId) return;
      
      console.log('Обратный отсчет до начала викторины:', data.secondsLeft);
      // Обновляем время в Redux для отображения обратного отсчета
      dispatch(updateRemainingTime(data.secondsLeft));
    }));
    
    // Обработчик нового вопроса
    handlers.push(wsClient.addMessageHandler<QuestionStartEvent>(WebSocketEventType.QUESTION_START, (data: QuestionStartEvent) => {
      // Если прислали вопрос для другой викторины, игнорируем
      if (data.quizId !== quizId) return;
      
      dispatch(setCurrentQuestion({
        id: data.questionId,
        quizId: data.quizId,
        text: data.text,
        type: 'single_choice',
        points: 10,
        timeLimitSeconds: data.durationSeconds,
        options: data.options.map((option) => ({
          id: option.id,
          text: option.text
        })),
        // Новые поля из обновленного интерфейса
        questionNumber: data.questionNumber,
        durationSeconds: data.durationSeconds,
        totalQuestions: data.totalQuestions,
        startTime: data.startTime
      }));
    }));
    
    // Обработчик таймера вопроса
    handlers.push(wsClient.addMessageHandler<QuizTimerEvent>(WebSocketEventType.QUIZ_TIMER, (data: QuizTimerEvent) => {
      // Если прислали таймер для другой викторины, игнорируем
      if (data.quizId !== quizId) return;
      
      dispatch(updateRemainingTime(data.remainingSeconds));
    }));
    
    // Обработчик завершения вопроса
    handlers.push(wsClient.addMessageHandler<QuestionEndEvent>(WebSocketEventType.QUESTION_END, (data: QuestionEndEvent) => {
      // Если прислали вопрос для другой викторины, игнорируем
      if (data.quizId !== quizId) return;
      
      dispatch(endCurrentQuestion({
        questionId: data.questionId,
        correctAnswer: data.correctOptionId
      }));
    }));
    
    // Обработчик показа правильного ответа
    handlers.push(wsClient.addMessageHandler<QuizAnswerRevealEvent>(WebSocketEventType.QUIZ_ANSWER_REVEAL, (data) => {
      // Обработка события показа правильного ответа
      console.log('Показан правильный ответ:', data);
      // Можно обновить UI, выделив правильный ответ
    }));

    // Обработчик результата ответа пользователя
    handlers.push(wsClient.addMessageHandler<QuizAnswerResultEvent>(WebSocketEventType.QUIZ_ANSWER_RESULT, (data) => {
      // Обработка события результата ответа
      console.log('Результат вашего ответа:', data);
      
      // Обновляем ответ пользователя с полученным результатом
      dispatch(updateUserAnswer({
        questionId: data.questionId,
        optionId: data.yourAnswer,
        isCorrect: data.isCorrect,
        pointsEarned: data.pointsEarned,
        timeTakenMs: data.timeTakenMs
      }));
    }));
    
    // Обработчик обновления результатов
    handlers.push(wsClient.addMessageHandler<ResultUpdateEvent>(WebSocketEventType.RESULT_UPDATE, (data: ResultUpdateEvent) => {
      // Если прислали результаты для другой викторины, игнорируем
      if (data.quizId !== quizId) return;
      
      // Обновляем таблицу лидеров
      if (data.leaderboard) {
        const leaderboard = data.leaderboard.map((item) => ({
          id: 0, // ID будет назначен бэкендом
          userId: item.userId,
          quizId: data.quizId,
          username: item.username,
          score: item.score,
          totalPoints: item.score, // Используем score как total_points
          correctAnswers: 0, // Эти данные могут отсутствовать в leaderboard
          totalQuestions: 0, // Эти данные могут отсутствовать в leaderboard
          completionTimeMs: 0, // Устанавливаем значение по умолчанию
          rank: item.position,
          completedAt: new Date().toISOString()
        }));
        
        dispatch(setLeaderboard(leaderboard));
      }
      
      // Обновляем статистику пользователя
      if (data.userStats) {
        dispatch(setUserResults({
          id: 0, // ID будет назначен бэкендом
          userId: 0, // ID пользователя
          quizId: data.quizId,
          username: '', // Имя пользователя можно получить из auth store
          score: data.userStats.score,
          totalPoints: data.userStats.score, // Используем score как total_points
          correctAnswers: data.userStats.correctAnswers,
          totalQuestions: data.userStats.totalAnswers,
          completionTimeMs: 0, // Устанавливаем значение по умолчанию
          rank: data.userStats.position,
          completedAt: new Date().toISOString()
        }));
      }
    }));
    
    // Обработчик таблицы лидеров 
    handlers.push(wsClient.addMessageHandler<QuizLeaderboardEvent>(WebSocketEventType.QUIZ_LEADERBOARD, (data) => {
      // Если прислали результаты для другой викторины, игнорируем
      if (data.quizId !== quizId) return;
      
      // Преобразуем результаты в формат наших типов
      const leaderboard = data.results.map((item) => ({
        id: 0, // ID будет назначен бэкендом
        userId: item.userId,
        quizId: data.quizId,
        username: item.username,
        score: item.score,
        totalPoints: item.score, // Используем score как total_points
        correctAnswers: item.correctAnswers,
        totalQuestions: 0, // Эти данные могут отсутствовать в leaderboard
        completionTimeMs: 0, // Устанавливаем значение по умолчанию
        rank: item.rank,
        completedAt: new Date().toISOString()
      }));
      
      dispatch(setLeaderboard(leaderboard));
    }));
    
    // Обработчик завершения викторины
    handlers.push(wsClient.addMessageHandler<QuizEndEvent>(WebSocketEventType.QUIZ_END, (data: QuizEndEvent) => {
      // Если прислали сообщение для другой викторины, игнорируем
      if (data.quizId !== quizId) return;
      
      dispatch(setQuizEnded());
      
      // Если есть данные о победителях, можно также обновить лидерборд
      if (data.winners && data.winners.length > 0) {
        const leaderboard = data.winners.map((winner) => ({
          id: 0,
          userId: winner.userId,
          quizId: data.quizId,
          username: winner.username,
          score: winner.score,
          totalPoints: winner.score,
          correctAnswers: 0,
          totalQuestions: 0,
          completionTimeMs: 0,
          rank: winner.position,
          completedAt: new Date().toISOString()
        }));
        
        dispatch(setLeaderboard(leaderboard));
      }
    }));
    
    // Обработчик отмены викторины
    handlers.push(wsClient.addMessageHandler<QuizCancelledEvent>(WebSocketEventType.QUIZ_CANCELLED, (data: QuizCancelledEvent) => {
      // Если прислали сообщение для другой викторины, игнорируем
      if (data.quizId !== quizId) return;
      
      dispatch(setQuizCancelled());
    }));
    
    // Очищаем обработчики при размонтировании
    return () => {
      handlers.forEach(removeHandler => removeHandler());
    };
  }, [quizId, dispatch]);
  
  return {
    isConnected
  };
};

/**
 * Хук для отправки ответа пользователя
 */
export const useSubmitAnswer = () => {
  const [submitLoading, setSubmitLoading] = useState(false);
  const [submitError, setSubmitError] = useState<Error | null>(null);
  
  const submitAnswer = useCallback(async ({ quizId, questionId, answerId }: { quizId: number | string, questionId: number | string, answerId: string }) => {
    setSubmitLoading(true);
    setSubmitError(null);
    
    try {
      // Отправляем ответ через WebSocket
      const success = wsClient.sendMessage<UserAnswerEvent>(WebSocketEventType.USER_ANSWER, {
        quizId: Number(quizId),
        questionId: Number(questionId),
        optionId: Number(answerId),
        answerTime: new Date().toISOString()
      });
      
      if (!success) {
        throw new Error('Failed to send answer via WebSocket');
      }
      
      return true;
    } catch (error) {
      setSubmitError(error instanceof Error ? error : new Error('An unknown error occurred'));
      
      // Резервный вариант: попробовать отправить ответ через HTTP API
      try {
        await apiClient.post(`/quizzes/${quizId}/questions/${questionId}/answer`, {
          optionId: answerId
        });
        return true;
      } catch (httpError) {
        console.error('Failed to submit answer via HTTP fallback:', httpError);
        throw error; // Пробрасываем исходную ошибку
      }
    } finally {
      setSubmitLoading(false);
    }
  }, []);
  
  return {
    submitAnswer,
    submitLoading,
    submitError
  };
}; 