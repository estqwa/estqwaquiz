"use client";

import { useParams, useRouter } from 'next/navigation';
import { useEffect, useRef, useState } from 'react';
import { useAuth } from '../../../../lib/auth/auth-context';
import { ApiError } from '../../../../lib/api/http-client';
import { getQuizById, Quiz } from '../../../../lib/api/quizzes';
import { getWebSocketTicket } from '../../../../lib/api/auth';
import {
  WsServerMessage,
  QuizQuestionData,
  QuizAnswerResultData,
  WsUserReadyMessage,
  WsUserAnswerMessage,
  WsUserHeartbeatMessage,
  QuizStartData,
  QuizEliminationData,
  ErrorData,
  QuizCountdownData,
  QuizWaitingRoomData,
  QuizAnnouncementData,
  QuizFinishData,
  QuizResultsAvailableData,
  QuizTimerData,
  QuizAnswerRevealData,
  QuizUserReadyData,
  QuizEliminationReminderData,
  ServerHeartbeatData
} from '@/types/websocket';

// Компонент страницы
export default function LiveQuizPage() {
  const router = useRouter();
  const params = useParams();
  const quizId = params.quizId as string;
  const { isAuthenticated, user } = useAuth();
  
  // Состояние WebSocket соединения
  const [wsConnected, setWsConnected] = useState(false);
  const [wsMessages, setWsMessages] = useState<string[]>([]);
  const wsRef = useRef<WebSocket | null>(null);
  
  // Состояние для загрузки информации о викторине
  const [quiz, setQuiz] = useState<Quiz | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [wsTicket, setWsTicket] = useState<string | null>(null);

  // Состояние для управления интерфейсом викторины
  const [quizStatus, setQuizStatus] = useState<'waiting' | 'starting' | 'in_progress' | 'completed'>('waiting');
  const [currentQuestion, setCurrentQuestion] = useState<QuizQuestionData | null>(null);
  const [timeRemaining, setTimeRemaining] = useState<number>(0);
  const [questionTimerInterval, setQuestionTimerInterval] = useState<NodeJS.Timeout | null>(null);
  const [preQuizTimerInterval, setPreQuizTimerInterval] = useState<NodeJS.Timeout | null>(null);
  const [answerSelected, setAnswerSelected] = useState<number | null>(null);
  const [answerSubmitted, setAnswerSubmitted] = useState<boolean>(false);
  const [answerResult, setAnswerResult] = useState<QuizAnswerResultData | null>(null);
  const [isEliminated, setIsEliminated] = useState<boolean>(false);
  const [quizResults, setQuizResults] = useState<QuizFinishData | null>(null);
  const [revealedCorrectOption, setRevealedCorrectOption] = useState<number | null>(null);
  const [questionCount, setQuestionCount] = useState<number>(0);
  const [currentQuestionNumber, setCurrentQuestionNumber] = useState<number>(0);
  const [quizAnnouncement, setQuizAnnouncement] = useState<QuizAnnouncementData | null>(null);
  const [waitingRoomInfo, setWaitingRoomInfo] = useState<QuizWaitingRoomData | null>(null);
  const [showResultsButton, setShowResultsButton] = useState<boolean>(false);

  // Лимит попыток переподключения
  const MAX_RECONNECT_ATTEMPTS = 5;
  const [reconnectAttempts, setReconnectAttempts] = useState(0);

  // --- Timer Refs ---
  // Используем useRef для хранения ID интервалов, чтобы избежать их включения в зависимости useEffect
  const questionTimerIntervalRef = useRef<NodeJS.Timeout | null>(null);
  const preQuizTimerIntervalRef = useRef<NodeJS.Timeout | null>(null);

  // --- Helper function to clear pre-quiz timer ---
  const clearPreQuizTimer = () => {
    // Используем ref для очистки
    if (preQuizTimerIntervalRef.current) {
      clearInterval(preQuizTimerIntervalRef.current);
      preQuizTimerIntervalRef.current = null;
    }
  };

  // --- Helper function to start pre-quiz timer ---
  const startPreQuizTimer = (initialSeconds: number) => {
    clearPreQuizTimer(); // Clear any existing pre-quiz timer
    setTimeRemaining(initialSeconds); // Set initial time

    if (initialSeconds > 0) {
      // Сохраняем ID интервала в ref
      preQuizTimerIntervalRef.current = setInterval(() => {
        setTimeRemaining(prev => {
          if (prev <= 1) {
            clearPreQuizTimer(); // Stop timer when it reaches 0
            return 0;
          }
          return prev - 1;
        });
      }, 1000);
    }
  };

  // --- Helper function to clear question timer ---
  const clearQuestionTimer = () => {
    if (questionTimerIntervalRef.current) {
        clearInterval(questionTimerIntervalRef.current);
        questionTimerIntervalRef.current = null;
    }
  };

  // --- Helper function to start question timer ---
  const startQuestionTimer = (initialSeconds: number) => {
      clearQuestionTimer(); // Clear existing timer first
      setTimeRemaining(initialSeconds); // Set initial time

      if (initialSeconds > 0) {
          questionTimerIntervalRef.current = setInterval(() => {
              setTimeRemaining(prev => {
                  if (prev <= 1) {
                      clearQuestionTimer();
                      return 0;
                  }
                  return prev - 1;
              });
          }, 1000);
      } else {
        setTimeRemaining(0); // Ensure time is 0 if initialSeconds is not positive
      }
  };

  // Загружаем информацию о викторине
  useEffect(() => {
    const loadQuizDetails = async () => {
      try {
        setLoading(true);
        setError(null);
        
        const quizId = parseInt(params.quizId as string, 10);
        if (isNaN(quizId)) {
          setError('Некорректный ID викторины');
          return;
        }
        
        const quizData = await getQuizById(quizId);
        setQuiz(quizData);
        setQuestionCount(quizData.question_count);
      } catch (err) {
        console.error('Ошибка загрузки деталей викторины:', err);
        setError((err as ApiError).error || 'Ошибка при загрузке деталей викторины');
      } finally {
        setLoading(false);
      }
    };

    if (isAuthenticated) {
      loadQuizDetails();
    }
  }, [params.quizId, isAuthenticated]);

  // Получаем WebSocket тикет, когда пользователь аутентифицирован
  useEffect(() => {
    const fetchWsTicket = async () => {
      try {
        // Получаем специальный тикет для WebSocket
        const ticket = await getWebSocketTicket();
        setWsTicket(ticket);
        console.log('WebSocket тикет получен');
      } catch (err) {
        console.error('Ошибка получения WebSocket тикета:', err);
        setError((err as ApiError).error || 'Не удалось получить тикет для WebSocket');
      }
    };

    if (isAuthenticated) {
      fetchWsTicket();
    }
  }, [isAuthenticated]);

  // Устанавливаем WebSocket соединение когда есть тикет
  useEffect(() => {
    // Функция для подключения WebSocket
    const connectWebSocket = async () => {
      // Закрываем предыдущее соединение, если оно есть
      if (wsRef.current && wsRef.current.readyState !== WebSocket.CLOSED) {
        wsRef.current.close();
      }

      if (!wsTicket) {
        console.error('Нет WebSocket тикета для подключения');
        return;
      }

      try {
        // Получаем URL для WebSocket с использованием нужного хоста
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        // Используем тот же хост, что и для API, но с ws/wss протоколом
        const wsHost = process.env.NEXT_PUBLIC_API_URL?.replace(/^https?:\/\//, '') || 'localhost:8080';
        // Добавляем тикет как query параметр
        const wsUrl = `${protocol}//${wsHost}/ws?ticket=${wsTicket}`;
        
        console.log(`Подключение к WebSocket: ${wsUrl}`);
        
        // Создаем WebSocket соединение
        const ws = new WebSocket(wsUrl);
        wsRef.current = ws;
        
        // Обработчики событий WebSocket
        ws.onopen = () => {
          console.log('WebSocket соединение установлено');
          setWsConnected(true);
          setReconnectAttempts(0); // Сбрасываем счетчик попыток при успешном подключении
          setError(null); // Очищаем ошибки при успешном подключении
          
          // Отправляем событие готовности с ID викторины
          if (quizId) {
            const readyMessage: WsUserReadyMessage = {
              type: 'user:ready',
              data: { quiz_id: parseInt(quizId, 10) }
            };
            console.log('Отправка сообщения user:ready', readyMessage);
            ws.send(JSON.stringify(readyMessage));
          } else {
            console.error('quizId не определен при попытке отправить user:ready');
          }
        };
        
        ws.onmessage = (event) => {
          try {
            const message = JSON.parse(event.data) as WsServerMessage;
            console.log('Получено сообщение WebSocket:', message);
            
            // Добавляем сообщение в лог
            setWsMessages(prev => [...prev, JSON.stringify(message)]);
            
            // Обработка различных типов сообщений
            handleWebSocketMessage(message);
          } catch (error) {
            console.error('Ошибка обработки сообщения WebSocket:', error);
          }
        };
        
        ws.onerror = (event) => {
          console.error('Ошибка WebSocket:', event);
          setTemporaryError('Ошибка WebSocket соединения', 8000); // Показываем сообщение на 8 секунд
          setWsConnected(false);
        };
        
        ws.onclose = (event) => {
          const closeReasons: Record<number, string> = {
            1000: 'Нормальное закрытие',
            1001: 'Закрытие из-за ухода с страницы',
            1002: 'Закрытие из-за ошибки протокола',
            1003: 'Получены неприемлемые данные',
            1005: 'Закрытие без указания причины',
            1006: 'Аномальное закрытие соединения',
            1007: 'Недопустимые данные в сообщении',
            1008: 'Нарушение политики',
            1009: 'Сообщение слишком большое',
            1010: 'Необходимые расширения отсутствуют',
            1011: 'Ошибка на сервере',
            1012: 'Перезапуск сервера',
            1013: 'Попробуйте позже',
            1014: 'Ошибка прокси',
            1015: 'Ошибка TLS'
          };
          
          const reasonText = closeReasons[event.code] || `Неизвестная причина (код ${event.code})`;
          console.log(`WebSocket соединение закрыто: ${reasonText}`);
          setWsConnected(false);
          
          // Очистка ТАЙМЕРОВ при закрытии соединения - ПЕРЕМЕЩЕНО В ОТДЕЛЬНЫЕ USEEFFECT
          // if (questionTimerInterval) {
          //   clearInterval(questionTimerInterval);
          //   setQuestionTimerInterval(null);
          // }
          // clearPreQuizTimer(); // Clear pre-quiz timer too
          
          // Попытка переподключения при неожиданном закрытии
          if (event.code !== 1000 && event.code !== 1001) {
            if (reconnectAttempts < MAX_RECONNECT_ATTEMPTS) {
              console.log(`Попытка переподключения ${reconnectAttempts + 1} из ${MAX_RECONNECT_ATTEMPTS}...`);
              // Экспоненциальная задержка: чем больше попыток, тем больше задержка
              const delay = Math.min(30000, Math.pow(2, reconnectAttempts) * 1000 + Math.random() * 1000);
              console.log(`Следующая попытка через ${Math.round(delay / 1000)} сек.`);
              
              setError(`Соединение потеряно (${reasonText}). Попытка переподключения ${reconnectAttempts + 1} из ${MAX_RECONNECT_ATTEMPTS} через ${Math.round(delay / 1000)} сек...`);
              
              setTimeout(() => {
                setReconnectAttempts(prev => prev + 1);
                connectWebSocket();
              }, delay);
            } else {
              setError(`Превышено количество попыток переподключения. Пожалуйста, обновите страницу.`);
            }
          }
        };
      } catch (error) {
        console.error('Ошибка при установке WebSocket соединения:', error);
        setError('Не удалось установить соединение с сервером викторины');
      }
    };

    if (isAuthenticated && wsTicket && quizId) {
      connectWebSocket();
    }

    // Очистка при размонтировании компонента
    return () => {
      // Clear BOTH timers on unmount - ПЕРЕМЕЩЕНО В ОТДЕЛЬНЫЕ USEEFFECT
      // if (questionTimerInterval) {
      //   clearInterval(questionTimerInterval);
      // }
      // clearPreQuizTimer();

      if (wsRef.current) {
        console.log('Закрытие WebSocket соединения при размонтировании компонента');
        wsRef.current.close(1000, "Component unmounting"); // Добавляем код и причину
      }
    };
  }, [isAuthenticated, wsTicket, quizId, reconnectAttempts]);
  
  // --- useEffect for Pre-Quiz Timer Management ---
  useEffect(() => {
    if (quizStatus === 'waiting' && waitingRoomInfo) {
      startPreQuizTimer(waitingRoomInfo.starts_in_seconds);
    } else if (quizStatus !== 'starting') {
      clearPreQuizTimer();
    }

    // Cleanup function for this effect
    return () => {
      clearPreQuizTimer();
    };
  // Зависим от статуса и данных для ожидания/отсчета
  }, [quizStatus, waitingRoomInfo]);

  // --- useEffect for Question Timer Management ---
  useEffect(() => {
      if (quizStatus === 'in_progress' && currentQuestion && !answerSubmitted && revealedCorrectOption === null) {
          startQuestionTimer(currentQuestion.time_limit);
      } else {
          // Clear question timer if not in progress, no question, answer submitted, or answer revealed
          clearQuestionTimer();
      }

      // Cleanup function for this effect
      return () => {
          clearQuestionTimer();
      };
  // Зависим от статуса, вопроса, отправки ответа и показа правильного
  }, [quizStatus, currentQuestion, answerSubmitted, revealedCorrectOption]);

  // Обработчик WebSocket сообщений
  const handleWebSocketMessage = (message: WsServerMessage) => {
    switch (message.type) {
      // ----- Основной флоу викторины -----
      case 'quiz:start':
        handleQuizStart(message.data);
        break;
      case 'quiz:question':
        handleQuizQuestion(message.data);
        break;
      case 'quiz:timer':
        handleTimerUpdate(message.data);
        break;
      case 'quiz:answer_reveal':
        handleAnswerReveal(message.data);
        break;
      case 'quiz:answer_result':
        handleAnswerResult(message.data);
        break;
      case 'quiz:finish':
        handleQuizFinish(message.data);
        break;
      case 'quiz:results_available':
        handleResultsAvailable(message.data);
        break;
      
      // ----- События ДО старта викторины -----
      case 'quiz:announcement':
        handleAnnouncement(message.data);
        break;
      case 'quiz:waiting_room':
        handleWaitingRoom(message.data);
        break;
      case 'quiz:countdown':
        handleCountdown(message.data);
        break;

      // ----- Другие события -----
      case 'quiz:elimination':
        handleElimination(message.data);
        break;
      case 'quiz:elimination_reminder':
        handleEliminationReminder(message.data);
        break;
      case 'quiz:user_ready':
        handleUserReady(message.data);
        break;
      case 'server:heartbeat':
        break;
      case 'error':
        handleError(message.data);
        break;

      // ----- Устаревшие/Неподтвержденные типы -----
      /* case 'quiz:leaderboard': // Заменено на получение через REST?
        handleLeaderboardUpdate(message.data);
        break; */
      /* case 'quiz:end': // Заменено на quiz:finish
        handleQuizEnd(message.data);
        break; */
      
      default:
        const exhaustiveCheck: never = message;
        break;
    }
  };

  // Обработчик начала викторины
  const handleQuizStart = (data: QuizStartData) => {
    console.log("Обработка: quiz:start", data);
    clearPreQuizTimer(); // Stop pre-quiz timer
    setQuizStatus('in_progress');
    setCurrentQuestion(null);
    setAnswerResult(null);
    setIsEliminated(false);
    setQuizResults(null);
    setRevealedCorrectOption(null);
    setQuizAnnouncement(null);
    setWaitingRoomInfo(null);
    setShowResultsButton(false);
    if(data.question_count) {
        setQuestionCount(data.question_count);
    }
  };

  // Обработчик сообщения с вопросом
  const handleQuizQuestion = (data: QuizQuestionData) => {
    console.log("Обработка: quiz:question", data);
    clearPreQuizTimer(); // Stop pre-quiz timer if it was somehow running
    setQuizStatus('in_progress');
    setAnswerSelected(null);
    setAnswerSubmitted(false);
    setAnswerResult(null);
    setRevealedCorrectOption(null);
    setCurrentQuestion(data);
    setCurrentQuestionNumber(data.number);
    setTimeRemaining(data.time_limit);

    // Clear local question timer just in case (although it's not started here)
    clearQuestionTimer();
  };

  // Обработчик результата ответа
  const handleAnswerResult = (data: QuizAnswerResultData) => {
    console.log("Обработка: quiz:answer_result", data);
    setAnswerResult(data);
    if (data.is_eliminated) {
      setIsEliminated(true);
    }
    // Stop the QUESTION timer when result arrives - УПРАВЛЯЕТСЯ ЧЕРЕЗ USEEFFECT [answerSubmitted]
    // if (questionTimerInterval) {
    //   clearInterval(questionTimerInterval);
    //   setQuestionTimerInterval(null);
    // }
  };

  // Обработчик выбывания из викторины
  const handleElimination = (data: QuizEliminationData) => {
    console.log("Обработка: quiz:elimination", data);
    setIsEliminated(true);
    setTemporaryError(`Вы выбыли: ${data.reason || data.message}`, 15000);
  };

  // Обработчик завершения викторины
  const handleQuizFinish = (data: QuizFinishData) => {
    console.log("Обработка: quiz:finish", data);
    setQuizStatus('completed');
    setQuizResults(data);
    setCurrentQuestion(null);
    setAnswerResult(null);
    setAnswerSelected(null);
    setAnswerSubmitted(false);
    setRevealedCorrectOption(null);
    // Stop the QUESTION timer if it's still running - УПРАВЛЯЕТСЯ ЧЕРЕЗ USEEFFECT [quizStatus]
    // if (questionTimerInterval) {
    //   clearInterval(questionTimerInterval);
    //   setQuestionTimerInterval(null);
    // }
    clearPreQuizTimer(); // Ensure pre-quiz timer is stopped
    setTimeRemaining(0);
  };

  // Функция для отправки ответа на вопрос
  const submitAnswer = (selectedOptionId: number) => {
    if (!currentQuestion || answerSubmitted || isEliminated) return;
    
    setAnswerSelected(selectedOptionId);
    setAnswerSubmitted(true);
    
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      const message: WsUserAnswerMessage = {
        type: 'user:answer',
        data: {
          question_id: currentQuestion.question_id,
          selected_option: selectedOptionId,
          timestamp: Date.now()
        }
      };
      
      wsRef.current.send(JSON.stringify(message));
      console.log('Отправлен ответ:', message);
    } else {
      console.error('WebSocket соединение не установлено');
      setError('Ошибка соединения. Не удалось отправить ответ.');
    }
  };

  // Функция для отправки heartbeat
  const sendHeartbeat = () => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      const message: WsUserHeartbeatMessage = {
        type: 'user:heartbeat',
        data: {}
      };
      wsRef.current.send(JSON.stringify(message));
    }
  };

  // Отправляем heartbeat каждые 30 секунд
  useEffect(() => {
    const heartbeatInterval = setInterval(() => {
      if (wsConnected) {
        sendHeartbeat();
      }
    }, 30000);

    return () => clearInterval(heartbeatInterval);
  }, [wsConnected]);

  // Функция форматирования времени в формат мм:сс
  const formatTime = (seconds: number): string => {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
  };

  // --- Компоненты UI --- 

  // Компонент для отображения состояния ожидания начала викторины
  const WaitingComponent = () => (
    <div className="text-center py-12 bg-white rounded-lg shadow-md">
      {waitingRoomInfo ? (
        <>
          <h2 className="text-2xl font-bold mb-4">Зал ожидания: {waitingRoomInfo.title}</h2>
          <p className="text-gray-600 mb-2">{waitingRoomInfo.description}</p>
          <p className="text-xl font-bold text-blue-600 mb-6">
            Викторина начнется через: {formatTime(timeRemaining)}
          </p>
        </>
      ) : quizAnnouncement ? (
        <>
          <h2 className="text-2xl font-bold mb-4">Анонс: {quizAnnouncement.title}</h2>
          <p className="text-gray-600 mb-2">{quizAnnouncement.description}</p>
          <p className="text-xl font-bold text-blue-600 mb-6">
            Начало через: {quizAnnouncement.minutes_to_start} мин.
          </p>
        </>
      ) : (
        <>
          <h2 className="text-2xl font-bold mb-4">Ожидание викторины</h2>
          <p className="text-gray-600 mb-6">Подключение к викторине...</p>
        </>
      )}
      <div className="animate-pulse flex justify-center">
        <div className="bg-blue-500 h-4 w-4 rounded-full mr-1"></div>
        <div className="bg-blue-500 h-4 w-4 rounded-full mr-1 animation-delay-200"></div>
        <div className="bg-blue-500 h-4 w-4 rounded-full animation-delay-400"></div>
      </div>
    </div>
  );

  // Компонент для отображения состояния обратного отсчета
  const StartingComponent = () => (
    <div className="text-center py-12 bg-white rounded-lg shadow-md">
      <h2 className="text-2xl font-bold mb-4">Викторина скоро начнется!</h2>
      {timeRemaining > 0 ? (
        <div className="mb-8">
          <p className="text-gray-600 mb-2">Старт через:</p>
          <p className="text-4xl font-bold text-blue-600">{formatTime(timeRemaining)}</p>
        </div>
      ) : (
        <p className="text-xl text-green-600 mb-6">
          Загрузка первого вопроса...
        </p>
      )}
      <div className="animate-pulse flex justify-center">
        <div className="bg-green-500 h-4 w-4 rounded-full mr-1"></div>
        <div className="bg-green-500 h-4 w-4 rounded-full mr-1 animation-delay-200"></div>
        <div className="bg-green-500 h-4 w-4 rounded-full animation-delay-400"></div>
      </div>
    </div>
  );

  // Компонент для отображения вопроса
  const QuestionComponent = () => {
    if (!currentQuestion) return null;
    const isRevealPhase = !!revealedCorrectOption; // Фаза показа правильного ответа

    return (
      <div className="bg-white rounded-lg shadow-md p-6">
        <div className="flex justify-between mb-4">
          <span className="font-medium">
            Вопрос {currentQuestionNumber} из {questionCount}
          </span>
          <span className={`font-medium ${timeRemaining <= 10 && !isRevealPhase ? 'text-red-600 animate-pulse' : 'text-blue-600'}`}>
            Время: {formatTime(timeRemaining)}
          </span>
        </div>
        
        <div className="mb-6">
          <h2 className="text-xl font-bold mb-6">{currentQuestion.text}</h2>
          
          <div className="space-y-3">
            {currentQuestion.options.map((option, index) => {
              const isSelected = answerSelected === option.id;
              const isCorrect = isRevealPhase && option.id === revealedCorrectOption;
              // Неправильный показываем только если он был выбран И наступила фаза показа
              const isIncorrect = isRevealPhase && isSelected && option.id !== revealedCorrectOption;
              
              let className = 'w-full text-left p-4 rounded-md transition-all duration-300 border';
              
              // Стиль в фазе показа ответа
              if (isRevealPhase) {
                 className += ' cursor-default';
                 if(isCorrect) {
                    className += ' bg-green-100 border-green-400 text-green-800 font-bold';
                 } else if (isIncorrect) {
                    className += ' bg-red-100 border-red-400 opacity-70';
                 } else {
                    className += ' bg-gray-50 border-gray-200 opacity-60';
                 }
              } 
              // Стиль ДО показа ответа (или если ответ еще не показан)
              else if (answerSubmitted || isEliminated || timeRemaining === 0) {
                 className += ' opacity-70 cursor-not-allowed bg-gray-100 border-gray-300';
                 if(isSelected) className += ' border-2 border-gray-400'; // Выбранный, но неактивный
              } 
              // Стиль активного выбора
              else {
                 className += ' cursor-pointer border-gray-200 hover:bg-gray-100';
                 if (isSelected) className += ' bg-blue-100 border-blue-500 border-2';
              }

              return (
                <button
                  key={option.id}
                  onClick={() => !answerSubmitted && !isEliminated && timeRemaining > 0 && !isRevealPhase && submitAnswer(option.id)}
                  disabled={answerSubmitted || isEliminated || timeRemaining === 0 || isRevealPhase}
                  className={className}
                >
                  <span className="font-medium">{String.fromCharCode(65 + index)}.</span> {option.text}
                </button>
              );
            })}
          </div>
        </div>
        
        {answerSubmitted && !answerResult && !isRevealPhase && (
          <div className="text-center text-green-600 font-medium">
            Ответ отправлен, ожидайте результатов...
          </div>
        )}
        
        {timeRemaining === 0 && !answerSubmitted && !isRevealPhase && (
          <div className="text-center text-red-600 font-medium">
            Время истекло!
          </div>
        )}
        
        {isEliminated && (
          <div className="mt-4 p-3 bg-red-50 border border-red-200 rounded-md text-red-600 text-center font-medium">
            Вы выбыли из викторины и можете только наблюдать.
          </div>
        )}
      </div>
    );
  };

  // Компонент для отображения результата ответа
  const AnswerResultComponent = () => {
    if (!answerResult || !currentQuestion) return null;
    
    // Показываем этот компонент только если правильный ответ еще НЕ показан (revealedCorrectOption === null)
    if(revealedCorrectOption !== null) return null;

    const userAnswerOption = currentQuestion.options.find(opt => opt.id === answerResult.your_answer);
    const userAnswerText = userAnswerOption?.text || (answerResult.time_limit_exceeded ? 'Время истекло' : 'Нет ответа');
    const userAnswerLetter = userAnswerOption ? String.fromCharCode(65 + currentQuestion.options.indexOf(userAnswerOption)) : '-';
    
    return (
      <div className={`mt-4 p-4 rounded-md ${answerResult.is_correct ? 'bg-green-50 border border-green-200' : 'bg-red-50 border border-red-200'}`}>
        <h3 className={`text-lg font-bold ${answerResult.is_correct ? 'text-green-600' : 'text-red-600'}`}> 
          {answerResult.is_correct ? '✅ Правильно!' : '❌ Неправильно!'}
        </h3>
        
        <div className="mt-2 space-y-1">
          <p>
            <span className="font-medium">Ваш ответ:</span>{' '}
            {userAnswerLetter}. {userAnswerText}
          </p>
          <p>
            <span className="font-medium">Очки за раунд:</span>{' '}
            {answerResult.points_earned}
          </p>
          <p>
            <span className="font-medium">Время ответа:</span>{' '}
            {(answerResult.time_taken_ms / 1000).toFixed(2)} сек
          </p>
          {answerResult.time_limit_exceeded && !answerResult.is_correct && <p className="text-orange-600">Время на ответ истекло!</p> }
        </div>
        {answerResult.is_eliminated && !isEliminated && <p className="text-red-600 font-medium mt-2">Вы были исключены!</p> }
      </div>
    );
  };

  const QuizResultsComponent = () => {
    if (!quizResults) return <div>Викторина завершена. Загрузка информации...</div>;
    return (
      <div className="text-center py-12 bg-white rounded-lg shadow-md">
        <h2 className="text-2xl font-bold mb-4">Викторина "{quizResults.title}" завершена!</h2>
        <p className="text-gray-600 mb-6">{quizResults.message}</p>
        {showResultsButton ? (
          <button 
            className="mt-4 p-2 bg-blue-500 text-white rounded hover:bg-blue-600"
            onClick={() => router.push(`/quiz/${quizResults.quiz_id}/results`)}
          >
            Посмотреть детальные результаты
          </button>
        ) : (
          <p className="text-gray-500">Ожидание публикации результатов...</p>
        )}
      </div>
    );
  };

  // Функция для установки временных сообщений об ошибках
  const setTemporaryError = (message: string, duration: number = 5000) => {
    setError(message);
    
    // Очистить сообщение через указанное время
    setTimeout(() => {
      setError(null);
    }, duration);
  };

  // --- НОВЫЕ Заглушки для обработчиков WS сообщений ---
  const handleTimerUpdate = (data: QuizTimerData) => {
    // This handles the QUESTION timer updates from the server
    // It might conflict slightly with the local interval, but server should be authoritative
    if (quizStatus === 'in_progress' && currentQuestion?.question_id === data.question_id && !answerSubmitted) {
      // Clear local timer if running, let server dictate
       clearQuestionTimer(); // Используем новую функцию очистки
       setTimeRemaining(data.remaining_seconds);
       // Optionally restart local timer if needed, or just rely on server updates
       // Если хотим полностью полагаться на сервер, можно не перезапускать локальный таймер
       // startQuestionTimer(data.remaining_seconds); // Опционально: перезапуск локального таймера
    }
  };

  const handleAnswerReveal = (data: QuizAnswerRevealData) => {
    console.log("Обработка: quiz:answer_reveal", data);
    setRevealedCorrectOption(data.correct_option);
    // Stop question timer when answer is revealed - УПРАВЛЯЕТСЯ ЧЕРЕЗ USEEFFECT [revealedCorrectOption]
    // if (questionTimerInterval) {
    //   clearInterval(questionTimerInterval);
    //   setQuestionTimerInterval(null);
    // }
  };

  const handleResultsAvailable = (data: QuizResultsAvailableData) => {
    console.log("Обработка: quiz:results_available", data);
    setShowResultsButton(true);
  };

  const handleAnnouncement = (data: QuizAnnouncementData) => {
    console.log("Обработка: quiz:announcement", data);
    setQuizAnnouncement(data);
    // We don't start a timer here, only display info
    clearPreQuizTimer(); // Clear any running pre-quiz timer
    setTimeRemaining(0); // Reset time as announcement doesn't have a precise second countdown
    setQuizStatus('waiting'); // Ensure status is waiting
  };

  const handleWaitingRoom = (data: QuizWaitingRoomData) => {
    console.log("Обработка: quiz:waiting_room", data);
    setWaitingRoomInfo(data);
    setQuizStatus('waiting');
    setQuizAnnouncement(null); // Clear announcement if waiting room info received
    // startPreQuizTimer(data.starts_in_seconds); // Управляется через useEffect [waitingRoomInfo]
  };

  const handleCountdown = (data: QuizCountdownData) => {
    console.log("Обработка: quiz:countdown", data);
    setQuizStatus('starting');
    setWaitingRoomInfo(null); // Clear waiting room info
    setQuizAnnouncement(null); // Clear announcement info
    startPreQuizTimer(data.seconds_left); // Start/Reset local timer (этот вызов оставим, т.к. countdown может приходить несколько раз)
  };
  
  const handleEliminationReminder = (data: QuizEliminationReminderData) => {
    console.log("Обработка: quiz:elimination_reminder", data);
    setTemporaryError(data.message, 10000);
    setIsEliminated(true);
  };

  const handleUserReady = (data: QuizUserReadyData) => {
    console.log("Обработка: quiz:user_ready", data);
  };

  const handleError = (data: ErrorData) => {
    console.error('Обработка: error', data);
    const errorMessage = data.message || 'Неизвестная ошибка от сервера';
    if (data.critical) {
      setError(`Критическая ошибка: ${errorMessage}`);
      setQuizStatus('waiting'); // Reset status on critical error
      // Clear BOTH timers - УПРАВЛЯЕТСЯ ЧЕРЕЗ ОТДЕЛЬНЫЕ USEEFFECT
      // if (questionTimerInterval) {
      //   clearInterval(questionTimerInterval);
      //   setQuestionTimerInterval(null);
      // }
      // clearPreQuizTimer();
    } else {
      setTemporaryError(`Ошибка: ${errorMessage}`, 10000);
    }
  };

  return (
    <div className="space-y-8">
      <h1 className="text-3xl font-bold">
        {loading ? 'Загрузка...' : quiz ? `Викторина: ${quiz.title}` : 'Активная викторина'}
      </h1>
      
      {loading && (
        <div className="flex justify-center my-12">
          <div className="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-blue-500"></div>
        </div>
      )}
      
      {error && (
        <div className="bg-red-50 text-red-600 p-4 rounded-md mb-6">
          {error}
        </div>
      )}
      
      {!loading && (
        <div className="space-y-6">
          {/* Индикатор подключения */}
          <div className="bg-white p-4 rounded-lg shadow-md">
            {wsConnected ? (
              <div className="flex items-center text-green-600 font-medium">
                <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5 mr-2" viewBox="0 0 20 20" fill="currentColor">
                  <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
                </svg>
                Соединение установлено
              </div>
            ) : (
              <div className="flex items-center text-red-600 font-medium">
                <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5 mr-2" viewBox="0 0 20 20" fill="currentColor">
                  <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
                </svg>
                Ожидание соединения...
              </div>
            )}
          </div>
          
          {/* Основной контент в зависимости от состояния викторины */}
          {quizStatus === 'waiting' && <WaitingComponent />}
          
          {quizStatus === 'starting' && <StartingComponent />}
          
          {/* Компоненты вопроса, ответа и лидерборда показываются только если викторина активна */}
          {quizStatus === 'in_progress' && (
            <>
              {currentQuestion && <QuestionComponent />}
              {answerResult && !revealedCorrectOption && <AnswerResultComponent />}
            </>
          )}
          
          {/* Компонент результатов показывается только если викторина завершена */}
          {quizStatus === 'completed' && <QuizResultsComponent />}
          
          {/* Дебаг-информация */}
          <div className="p-4 bg-gray-50 rounded-md mt-8 hidden">
            <h3 className="font-medium mb-2">Последние сообщения от сервера:</h3>
            {wsMessages.length > 0 ? (
              <ul className="text-sm text-gray-600 space-y-1 max-h-40 overflow-y-auto">
                {wsMessages.slice(-5).map((msg, index) => (
                  <li key={index} className="border-b border-gray-200 pb-1">{msg}</li>
                ))}
              </ul>
            ) : (
              <p className="text-sm text-gray-500">Ожидание сообщений...</p>
            )}
          </div>
        </div>
      )}
    </div>
  );
} 