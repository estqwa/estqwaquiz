import { useEffect } from 'react';
import { AppProps } from 'next/app';
import { Provider } from 'react-redux';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ReactQueryDevtools } from '@tanstack/react-query-devtools';
import { store } from '../store';
import { wsClient } from '../api/websocket/client';
import { useAppSelector } from '../hooks/redux-hooks';
import '../../styles/globals.css';
import { setActiveQuiz, setCurrentQuestion, updateRemainingTime } from '../store/quiz/slice';
import ErrorBoundary from '../components/ErrorBoundary';
import { authService } from '../api/services/authService';
import { 
  WebSocketEventType, 
  QuizStartEvent, 
  QuestionStartEvent, 
  QuizTimerEvent,
  QuizCountdownEvent
} from '../types/websocket';

// Создаем QueryClient один раз
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1, // Повторять запросы 1 раз при ошибке
      refetchOnWindowFocus: true, // Обновлять данные при фокусе на окне
    },
  },
});

// Компонент для управления WebSocket соединением
const WebSocketManager = () => {
  // Получаем статус аутентификации и флаг подключения WS из Redux
  const isAuthenticated = useAppSelector(state => state.auth.isAuthenticated);
  const isConnected = useAppSelector(state => state.websocket.isConnected);
  const isConnecting = useAppSelector(state => state.websocket.isConnecting);
  const useCookieAuth = useAppSelector(state => state.auth.useCookieAuth);

  useEffect(() => {
    let unsubscribeHandlers: (() => void)[] = [];

    if (isAuthenticated && !isConnected && !isConnecting) {
      console.log('User authenticated, attempting to connect WebSocket...');
      
      // Попытка подключения WebSocket
      const connectWebSocket = async () => {
        try {
          // Для режима Cookie Auth нужно получить WS-тикет через специальный эндпоинт
          if (useCookieAuth) {
            console.log('Getting WebSocket ticket for Cookie-Auth mode...');
            try {
              const { ticket } = await authService.getWsTicket();
              console.log('WebSocket ticket received successfully');
              await wsClient.connect(ticket);
            } catch (ticketError) {
              console.error('Failed to get WebSocket ticket:', ticketError);
              return; // Прерываем выполнение, если не удалось получить тикет
            }
          } else {
            // Для режима Bearer Auth продолжаем использовать token из Redux Store
            await wsClient.connect(); // connect без параметров использует token из store
          }
          
          console.log('WebSocket connected successfully via Manager.');
          
          // Регистрируем обработчики WS сообщений
          const unsubQuizStart = wsClient.addMessageHandler(WebSocketEventType.QUIZ_START, handleQuizStart);
          const unsubQuestionStart = wsClient.addMessageHandler(WebSocketEventType.QUESTION_START, handleQuestionStart);
          const unsubQuestionTimer = wsClient.addMessageHandler(WebSocketEventType.QUIZ_TIMER, handleQuestionTimer);
          const unsubQuizCountdown = wsClient.addMessageHandler(WebSocketEventType.QUIZ_COUNTDOWN, handleQuizCountdown);
          
          // Сохраняем функции отписки для дальнейшего использования
          unsubscribeHandlers = [unsubQuizStart, unsubQuestionStart, unsubQuestionTimer, unsubQuizCountdown];
        } catch (error) {
          console.error('WebSocket connection failed:', error);
          // Ошибка уже должна быть в Redux store через wsError
        }
      };

      // Вызываем функцию подключения
      connectWebSocket();
    } else if (!isAuthenticated && isConnected) {
      // Если пользователь вышел, а соединение еще активно, отключаемся
      console.log('User logged out, disconnecting WebSocket...');
      wsClient.disconnect();
    }

    // Функция очистки при размонтировании компонента или изменении isAuthenticated
    return () => {
      // Отписываемся от всех обработчиков сообщений
      unsubscribeHandlers.forEach(unsubscribe => unsubscribe());
      unsubscribeHandlers = [];
    };
  }, [isAuthenticated, isConnected, isConnecting, useCookieAuth]);

  // Обработчики сообщений WebSocket
  // Определены вне useEffect для избежания пересоздания при каждом рендере
  const handleQuizStart = (data: QuizStartEvent) => {
    store.dispatch(setActiveQuiz({
      quiz: {
        id: data.quizId,
        title: data.title,
        description: data.description || '',
        questionCount: data.numQuestions,
        status: 'active',
        startTime: data.startTime,
        durationMinutes: data.durationMinutes,
        // Добавляем обязательные поля для типа Quiz
        category: '',
        difficulty: 'medium',
        creatorId: 0,
        isPublic: true,
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString()
      }
    }));
  };
  
  const handleQuestionStart = (data: QuestionStartEvent) => {
    store.dispatch(setCurrentQuestion({
      id: data.questionId,
      quizId: data.quizId,
      text: data.text,
      type: 'single_choice',
      points: 10,
      timeLimitSeconds: data.durationSeconds,
      options: data.options.map((option: { id: number; text: string }) => ({
        id: option.id,
        text: option.text
      })),
      correctAnswers: [],
      orderNum: data.questionNumber,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString()
    }));
  };
  
  const handleQuestionTimer = (data: QuizTimerEvent) => {
    // Проверяем, относится ли таймер к текущей викторине
    const currentQuiz = store.getState().quiz.activeQuiz;
    if (currentQuiz && currentQuiz.id === data.quizId) {
      store.dispatch(updateRemainingTime(data.remainingSeconds));
    }
  };

  const handleQuizCountdown = (data: QuizCountdownEvent) => {
    // Обработка обратного отсчета до начала викторины
    const currentQuiz = store.getState().quiz.activeQuiz;
    if (currentQuiz && currentQuiz.id === data.quizId) {
      store.dispatch(updateRemainingTime(data.secondsLeft));
    }
  };

  // Этот компонент ничего не рендерит
  return null;
};

function MyApp({ Component, pageProps }: AppProps<{ dehydratedState: unknown }>) {
  return (
    <ErrorBoundary>
      <Provider store={store}>
        <QueryClientProvider client={queryClient}>
          {/* Основной компонент страницы */}
          <Component {...pageProps} />
          {/* Компонент для управления WS */}
          <WebSocketManager />
          {/* Инструменты разработчика React Query */}
          <ReactQueryDevtools initialIsOpen={false} />
        </QueryClientProvider>
      </Provider>
    </ErrorBoundary>
  );
}

export default MyApp; 