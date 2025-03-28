import React, { useEffect, useMemo } from 'react';
import { NextPage, GetServerSideProps } from 'next';
import { useRouter } from 'next/router';
import dynamic from 'next/dynamic';
import Layout from '../../../components/layout/Layout';
import Button from '../../../components/common/Button';
import ApiErrorAlert from '../../../components/common/ApiErrorAlert';
import { useQuiz } from '../../../hooks/query-hooks/useQuizQueries';
import { useAppSelector, useAppDispatch } from '../../../hooks/redux-hooks';
import { resetQuizState, setActiveQuiz } from '../../../store/quiz/slice';
import Link from 'next/link';
import ErrorBoundary from '../../../components/ErrorBoundary';

// Динамически импортируем компонент ActiveQuiz
const DynamicActiveQuiz = dynamic(
  () => import('../../../components/quiz/ActiveQuiz'),
  {
    loading: () => (
      <div className="flex justify-center items-center h-64">
        <div className="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-blue-500"></div>
      </div>
    ),
    ssr: false
  }
);

interface PlayQuizPageProps {
  id: string;
}

const PlayQuizPage: NextPage<PlayQuizPageProps> = ({ id }) => {
  const router = useRouter();
  const dispatch = useAppDispatch();
  const numericId = parseInt(id, 10);
  
  // Получаем данные о викторине (включая статус) из React Query
  const { data: quiz, isLoading, isError, error } = useQuiz(numericId);
  
  // Получаем состояние активной викторины из Redux
  const { activeQuiz, quizStatus } = useAppSelector(state => state.quiz);
  
  // Инициализируем состояние викторины
  useEffect(() => {
    if (quiz && !activeQuiz) {
      // Устанавливаем викторину в Redux store
      dispatch(setActiveQuiz({
        quiz,
        questions: [] // Вопросы будут загружены через WebSocket
      }));
    }
    
    // Очищаем состояние при размонтировании компонента
    return () => {
      dispatch(resetQuizState());
    };
  }, [quiz, activeQuiz, dispatch]);
  
  // Мемоизированная функция для проверки доступа к викторине
  const canPlayQuiz = useMemo(() => {
    if (!quiz) return false;
    
    // Проверяем, что викторина активна
    if (quiz.status !== 'active') {
      return false;
    }
    
    // Дополнительные проверки (при необходимости):
    // - Является ли пользователь участником
    // - Не исключен ли пользователь
    // - и т.д.
    
    return true;
  }, [quiz]);
  
  // Если данные загружаются, отображаем индикатор загрузки
  if (isLoading) {
    return (
      <Layout title="Загрузка викторины...">
        <div className="flex justify-center items-center h-64">
          <div className="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-blue-500"></div>
        </div>
      </Layout>
    );
  }
  
  // Если произошла ошибка, отображаем сообщение об ошибке
  if (isError || !quiz) {
    return (
      <Layout title="Ошибка">
        <ApiErrorAlert 
          error={error} 
          className="mb-6"
          onRetry={() => router.push('/quizzes')}
        />
        <div className="mt-6">
          <Link href="/quizzes">
            <Button variant="outline">Вернуться к списку викторин</Button>
          </Link>
        </div>
      </Layout>
    );
  }
  
  // Проверяем, может ли пользователь проходить викторину
  if (!canPlayQuiz) {
    return (
      <Layout title={quiz.title}>
        <div className="bg-yellow-50 p-6 rounded-lg">
          <h3 className="text-lg font-medium text-yellow-800 mb-2">Викторина недоступна</h3>
          <p className="text-sm text-yellow-600 mb-4">
            Эта викторина в данный момент не активна или вы не являетесь участником.
          </p>
          <div className="flex space-x-4">
            <Link href={`/quizzes/${id}`}>
              <Button variant="outline">Вернуться к странице викторины</Button>
            </Link>
            <Link href="/quizzes">
              <Button variant="outline">К списку викторин</Button>
            </Link>
          </div>
        </div>
      </Layout>
    );
  }
  
  // Рендеринг страницы активной викторины
  return (
    <Layout title={quiz.title}>
      <ErrorBoundary>
        <div className="mb-4">
          <Link href={`/quizzes/${id}`} className="text-blue-600 hover:text-blue-800 flex items-center">
            <svg className="h-5 w-5 mr-1" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
              <path fillRule="evenodd" d="M12.707 5.293a1 1 0 010 1.414L9.414 10l3.293 3.293a1 1 0 01-1.414 1.414l-4-4a1 1 0 010-1.414l4-4a1 1 0 011.414 0z" clipRule="evenodd" />
            </svg>
            Назад к странице викторины
          </Link>
        </div>
        
        {/* Компонент ActiveQuiz отвечает за управление состоянием активной викторины через WebSocket */}
        <DynamicActiveQuiz quizId={numericId} />
      </ErrorBoundary>
    </Layout>
  );
};

export const getServerSideProps: GetServerSideProps = async (context) => {
  const { id } = context.params as { id: string };
  
  return {
    props: {
      id
    }
  };
};

export default PlayQuizPage; 