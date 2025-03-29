import { useEffect } from 'react';
import { AppProps } from 'next/app';
import { Provider } from 'react-redux';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ReactQueryDevtools } from '@tanstack/react-query-devtools';
import { store } from '../store';
import { wsClient } from '../api/websocket/client';
import { useAppSelector, useAppDispatch } from '../hooks/redux-hooks';
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
import { loginSuccess, setAuthStatus, setAuthChecked } from '../store/auth/slice';

// Создаем QueryClient один раз
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1, // Повторять запросы 1 раз при ошибке
      refetchOnWindowFocus: true, // Обновлять данные при фокусе на окне
    },
  },
});

// Компонент для проверки авторизации при загрузке приложения
const AuthInitializer = () => {
  const dispatch = useAppDispatch();
  const authStatus = useAppSelector(state => state.auth.status);
  
  useEffect(() => {
    // Проверяем статус авторизации только один раз при монтировании компонента
    if (authStatus === 'idle') {
      const checkAuthStatus = async () => {
        dispatch(setAuthStatus('loading'));
        
        try {
          // Проверяем авторизацию, запрашивая текущего пользователя
          const userData = await authService.getCurrentUser();
          
          // Если запрос успешный, устанавливаем данные пользователя
          if (userData) {
            dispatch(loginSuccess({ 
              user: userData, 
              csrfToken: null // Здесь можно обновить, если бэкенд вернет CSRF-токен
            }));
          }
        } catch (error) {
          // При ошибке (401) устанавливаем статус 'failed'
          console.error('Failed to initialize auth:', error);
          dispatch(setAuthStatus('failed'));
        } finally {
          // В любом случае отмечаем, что проверка завершена
          dispatch(setAuthChecked());
        }
      };
      
      checkAuthStatus();
    }
  }, [dispatch, authStatus]);
  
  // Этот компонент ничего не рендерит
  return null;
};

// Компонент для управления WebSocket соединением
const WebSocketManager = () => {
  // Получаем статус аутентификации и флаг подключения WS из Redux
  const isAuthenticated = useAppSelector(state => state.auth.isAuthenticated);
  const isConnected = useAppSelector(state => state.websocket.isConnected);
  const isConnecting = useAppSelector(state => state.websocket.isConnecting);
  const authChecked = useAppSelector(state => state.auth.status === 'checked' || state.auth.status === 'succeeded');

  useEffect(() => {
    let unsubscribeHandlers: (() => void)[] = [];

    // Подключаемся к WebSocket только когда пользователь аутентифицирован и проверка авторизации завершена
    if (isAuthenticated && authChecked && !isConnected && !isConnecting) {
      console.log('User authenticated, attempting to connect WebSocket...');
      
      // Попытка подключения WebSocket
      const connectWebSocket = async () => {
        try {
          // Получаем WebSocket-тикет
          const { ticket } = await authService.getWsTicket();
          console.log('WebSocket ticket received successfully');
          await wsClient.connect(ticket);
          
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
  }, [isAuthenticated, isConnected, isConnecting, authChecked]);

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
    console.log('Получен QUIZ_COUNTDOWN в _app.tsx:', data);
    
    // Получаем текущее состояние викторины
    const quizState = store.getState().quiz;
    console.log('Текущее состояние quiz:', { 
      activeQuiz: quizState.activeQuiz?.id, 
      quizStatus: quizState.quizStatus 
    });
    
    // Обновляем таймер даже если викторина еще не активна или мы еще не знаем quizId
    // Это важно для отображения обратного отсчета перед началом викторины
    store.dispatch(updateRemainingTime(data.secondsLeft));
    
    // Если у нас нет активной викторины, но пришел обратный отсчет, можно установить активную викторину
    if (!quizState.activeQuiz && data.quizId) {
      console.log('Устанавливаем начальную информацию о викторине из countdown');
      store.dispatch(setActiveQuiz({
        quiz: {
          id: data.quizId,
          title: 'Викторина',  // Временное название до получения полных данных
          description: 'Загрузка данных викторины...',
          status: 'published',  // Используем published вместо waiting
          questionCount: 0,    // Будет обновлено позже при получении quiz:start
          startTime: new Date().toISOString(), // Текущее время как временное значение
          // Дополнительные обязательные поля
          category: '',
          difficulty: 'medium',
          creatorId: 0,
          isPublic: true,
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString()
        }
      }));
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
          {/* Инициализация авторизации */}
          <AuthInitializer />
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