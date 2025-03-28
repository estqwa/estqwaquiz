import React, { useEffect } from 'react';
import { NextPage } from 'next';
import { useRouter } from 'next/router';
import dynamic from 'next/dynamic';
import Layout from '../components/layout/Layout';
import Button from '../components/common/Button';
import ApiErrorAlert from '../components/common/ApiErrorAlert';
import { useActiveQuiz, useNextScheduledQuiz } from '../hooks/query-hooks/useQuizQueries';
import { useAppSelector, useAppDispatch } from '../hooks/redux-hooks';
import { resetQuizState, setActiveQuiz } from '../store/quiz/slice';
import Link from 'next/link';
import ErrorBoundary from '../components/ErrorBoundary';

// Динамически импортируем компонент ActiveQuiz
const DynamicActiveQuiz = dynamic(
  () => import('../components/quiz/ActiveQuiz'),
  {
    loading: () => (
      <div className="flex justify-center items-center h-64">
        <div className="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-blue-500"></div>
      </div>
    ),
    ssr: false
  }
);

const PlayPage: NextPage = () => {
  const router = useRouter();
  const dispatch = useAppDispatch();
  
  // Получаем данные о текущей активной викторине
  const { data: activeQuiz, isLoading: isLoadingActive, isError: isErrorActive, error: errorActive } = useActiveQuiz();
  
  // Получаем данные о ближайшей запланированной викторине
  const { data: nextScheduledQuiz, isLoading: isLoadingScheduled, isError: isErrorScheduled, error: errorScheduled } = useNextScheduledQuiz();
  
  // Получаем состояние активной викторины из Redux
  const { activeQuiz: activeQuizState } = useAppSelector(state => state.quiz);
  
  // Инициализируем состояние викторины
  useEffect(() => {
    if (activeQuiz && !activeQuizState) {
      // Устанавливаем викторину в Redux store
      dispatch(setActiveQuiz({
        quiz: activeQuiz
      }));
    }
    
    // Очищаем состояние при размонтировании компонента
    return () => {
      dispatch(resetQuizState());
    };
  }, [activeQuiz, activeQuizState, dispatch]);
  
  // Если данные загружаются, отображаем индикатор загрузки
  if (isLoadingActive || isLoadingScheduled) {
    return (
      <Layout title="Загрузка викторины...">
        <div className="flex justify-center items-center h-64">
          <div className="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-blue-500"></div>
        </div>
      </Layout>
    );
  }
  
  // Если произошла ошибка, отображаем сообщение об ошибке
  if ((isErrorActive || isErrorScheduled) && !activeQuiz && !nextScheduledQuiz) {
    return (
      <Layout title="Ошибка">
        <ApiErrorAlert 
          error={errorActive || errorScheduled} 
          className="mb-6"
          onRetry={() => router.push('/')}
        />
        <div className="mt-6">
          <Link href="/">
            <Button variant="outline">Вернуться на главную</Button>
          </Link>
        </div>
      </Layout>
    );
  }
  
  // Если есть активная викторина, показываем ее
  if (activeQuiz) {
    return (
      <Layout title={activeQuiz.title}>
        <ErrorBoundary>
          <div className="mb-4">
            <Link href="/" className="text-blue-600 hover:text-blue-800 flex items-center">
              <svg className="h-5 w-5 mr-1" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
                <path fillRule="evenodd" d="M12.707 5.293a1 1 0 010 1.414L9.414 10l3.293 3.293a1 1 0 01-1.414 1.414l-4-4a1 1 0 010-1.414l4-4a1 1 0 011.414 0z" clipRule="evenodd" />
              </svg>
              Назад на главную
            </Link>
          </div>
          
          {/* Компонент ActiveQuiz отвечает за управление состоянием активной викторины через WebSocket */}
          <DynamicActiveQuiz quizId={activeQuiz.id} />
        </ErrorBoundary>
      </Layout>
    );
  }
  
  // Если нет активной викторины, но есть запланированная, показываем информацию о ней
  if (nextScheduledQuiz) {
    const startTime = new Date(nextScheduledQuiz.start_time || nextScheduledQuiz.scheduled_time || '');
    const timeUntilStart = startTime.getTime() - Date.now();
    const hoursUntilStart = Math.floor(timeUntilStart / (1000 * 60 * 60));
    const minutesUntilStart = Math.floor((timeUntilStart % (1000 * 60 * 60)) / (1000 * 60));

    return (
      <Layout title="Ожидание викторины">
        <div className="bg-white p-8 rounded-lg shadow-md">
          <h2 className="text-2xl font-bold mb-4">Следующая викторина:</h2>
          <div className="mb-6">
            <h3 className="text-xl font-semibold">{nextScheduledQuiz.title}</h3>
            <p className="text-gray-600 mt-2">{nextScheduledQuiz.description}</p>
            <div className="mt-4 p-4 bg-blue-50 rounded-md">
              <p className="font-medium">Начало через: {hoursUntilStart}ч {minutesUntilStart}мин</p>
              <p>Дата и время начала: {startTime.toLocaleString()}</p>
              <p className="mt-2">Количество вопросов: {nextScheduledQuiz.question_count || 'Не указано'}</p>
              
              {nextScheduledQuiz.duration_minutes && (
                <p className="mt-1">Продолжительность: {nextScheduledQuiz.duration_minutes} минут</p>
              )}
              
              {nextScheduledQuiz.difficulty && (
                <p className="mt-1">Сложность: {
                  nextScheduledQuiz.difficulty === 'easy' ? 'Лёгкая' :
                  nextScheduledQuiz.difficulty === 'medium' ? 'Средняя' : 'Сложная'
                }</p>
              )}
            </div>
          </div>
          <div className="mt-8 space-y-4">
            <Button variant="primary" className="w-full" disabled>
              Ожидание начала викторины
            </Button>
            <p className="text-center text-gray-600 text-sm">
              Страница автоматически обновится, когда викторина начнется
            </p>
            <div className="text-center">
              <Link href="/" className="text-blue-600 hover:text-blue-800">
                Вернуться на главную
              </Link>
            </div>
          </div>
        </div>
      </Layout>
    );
  }
  
  // Если нет ни активной, ни запланированной викторины
  return (
    <Layout title="Нет доступных викторин">
      <div className="bg-white p-8 rounded-lg shadow-md text-center">
        <h2 className="text-2xl font-bold mb-4">В данный момент нет доступных викторин</h2>
        <p className="text-gray-600 mb-8">
          Сейчас нет активных или запланированных викторин. Пожалуйста, загляните позже.
        </p>
        <Link href="/">
          <Button variant="primary">Вернуться на главную</Button>
        </Link>
      </div>
    </Layout>
  );
};

export default PlayPage; 