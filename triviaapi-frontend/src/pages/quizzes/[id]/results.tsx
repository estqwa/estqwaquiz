import React, { useState } from 'react';
import { NextPage, GetServerSideProps } from 'next';
import { useRouter } from 'next/router';
import Layout from '../../../components/layout/Layout';
import Button from '../../../components/common/Button';
import { useQuiz, useQuizResults, useUserQuizResult } from '../../../hooks/query-hooks/useQuizQueries';
import { formatDateTime } from '../../../utils/date-utils';
import { useAppSelector } from '../../../hooks/redux-hooks';
import Link from 'next/link';

const QuizResultsPage: NextPage<{ id: string }> = ({ id }) => {
  const router = useRouter();
  const { isAuthenticated, user } = useAppSelector(state => state.auth);
  const [page, setPage] = useState(1);
  const [showUserResult, setShowUserResult] = useState(true);
  
  // Получаем данные о викторине
  const { data: quiz, isLoading: isQuizLoading } = useQuiz(id);
  
  // Получаем результаты всех участников
  const { data: resultsData, isLoading: isResultsLoading, isError, error } = useQuizResults(id, page, 10);
  
  // Получаем результаты текущего пользователя
  const { data: userResult, isLoading: isUserResultLoading } = useUserQuizResult(
    id,
    isAuthenticated && user ? user.id : null
  );
  
  const isLoading = isQuizLoading || isResultsLoading || isUserResultLoading;
  const totalPages = resultsData?.meta?.pagination?.last_page || 1;
  
  // Обработчики пагинации
  const goToPage = (newPage: number) => {
    setPage(newPage);
    window.scrollTo({ top: 0, behavior: 'smooth' });
  };
  
  // Если данные загружаются, отображаем индикатор загрузки
  if (isLoading) {
    return (
      <Layout title="Загрузка результатов...">
        <div className="flex justify-center items-center h-64">
          <div className="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-blue-500"></div>
        </div>
      </Layout>
    );
  }
  
  // Если произошла ошибка, отображаем сообщение об ошибке
  if (isError || !quiz || !resultsData) {
    return (
      <Layout title="Ошибка">
        <div className="bg-red-50 p-4 rounded-md mb-6">
          <div className="flex">
            <div className="flex-shrink-0">
              <svg className="h-5 w-5 text-red-400" viewBox="0 0 20 20" fill="currentColor">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
              </svg>
            </div>
            <div className="ml-3">
              <h3 className="text-sm font-medium text-red-800">
                Ошибка при загрузке результатов
              </h3>
              <div className="mt-2 text-sm text-red-700">
                <p>
                  {error instanceof Error ? error.message : 'Произошла ошибка при загрузке результатов. Попробуйте обновить страницу.'}
                </p>
              </div>
            </div>
          </div>
        </div>
        <div className="mt-6">
          <Link href={`/quizzes/${id}`}>
            <Button variant="outline">Вернуться к викторине</Button>
          </Link>
        </div>
      </Layout>
    );
  }
  
  return (
    <Layout title={`Результаты - ${quiz.title}`}>
      <div className="bg-white rounded-lg shadow-sm overflow-hidden">
        {/* Шапка результатов */}
        <div className="px-6 py-5 border-b border-gray-200">
          <div className="flex items-center space-x-3">
            <Link href={`/quizzes/${id}`} className="text-blue-600 hover:text-blue-800">
              <svg className="h-5 w-5" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 19l-7-7m0 0l7-7m-7 7h18" />
              </svg>
            </Link>
            <h1 className="text-2xl font-bold text-gray-900">{quiz.title} - Результаты</h1>
          </div>
          <p className="mt-2 text-sm text-gray-600">
            Завершено {formatDateTime(quiz.completed_at || quiz.updated_at)}
          </p>
        </div>
        
        {/* Результаты текущего пользователя */}
        {isAuthenticated && userResult && showUserResult && (
          <div className="px-6 py-4 bg-blue-50 border-b border-blue-100">
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-medium text-gray-900">Ваш результат</h2>
              <button 
                onClick={() => setShowUserResult(false)}
                className="text-sm text-gray-500 hover:text-gray-700"
              >
                Скрыть
              </button>
            </div>
            <div className="mt-4 grid grid-cols-1 sm:grid-cols-3 gap-4">
              <div className="bg-white p-4 rounded-lg shadow-sm">
                <p className="text-sm font-medium text-gray-500">Место</p>
                <p className="mt-1 text-2xl font-semibold text-gray-900">
                  {userResult.rank || '-'}
                </p>
              </div>
              <div className="bg-white p-4 rounded-lg shadow-sm">
                <p className="text-sm font-medium text-gray-500">Баллы</p>
                <p className="mt-1 text-2xl font-semibold text-gray-900">
                  {userResult.score} / {quiz.questions_count * 100}
                </p>
              </div>
              <div className="bg-white p-4 rounded-lg shadow-sm">
                <p className="text-sm font-medium text-gray-500">Правильные ответы</p>
                <p className="mt-1 text-2xl font-semibold text-gray-900">
                  {userResult.correct_answers} / {quiz.questions_count}
                </p>
              </div>
            </div>
          </div>
        )}
        
        {/* Таблица результатов */}
        <div className="px-6 py-5">
          <h2 className="text-lg font-medium text-gray-900 mb-4">Таблица лидеров</h2>
          
          {resultsData.data.length === 0 ? (
            <div className="text-center py-8">
              <p className="text-gray-500">Нет доступных результатов</p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                  <tr>
                    <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Ранг
                    </th>
                    <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Участник
                    </th>
                    <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Баллы
                    </th>
                    <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Правильные ответы
                    </th>
                    <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Время завершения
                    </th>
                  </tr>
                </thead>
                <tbody className="bg-white divide-y divide-gray-200">
                  {resultsData.data.map((result) => (
                    <tr 
                      key={result.id} 
                      className={isAuthenticated && user && result.user_id === user.id ? 'bg-blue-50' : ''}
                    >
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div className="text-sm font-medium text-gray-900">
                          {result.rank}
                        </div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div className="flex items-center">
                          <div className="h-8 w-8 rounded-full bg-gray-200 flex items-center justify-center">
                            <span className="text-sm font-medium text-gray-600">
                              {result.user.username.charAt(0).toUpperCase()}
                            </span>
                          </div>
                          <div className="ml-4">
                            <div className="text-sm font-medium text-gray-900">
                              {result.user.username}
                            </div>
                          </div>
                        </div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div className="text-sm text-gray-900">
                          {result.score} / {quiz.questions_count * 100}
                        </div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div className="text-sm text-gray-900">
                          {result.correct_answers} / {quiz.questions_count}
                        </div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                        {formatDateTime(result.completed_at)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
          
          {/* Пагинация */}
          {totalPages > 1 && (
            <div className="flex justify-center mt-8">
              <nav className="relative z-0 inline-flex rounded-md shadow-sm -space-x-px" aria-label="Pagination">
                <button
                  onClick={() => goToPage(page - 1)}
                  disabled={page === 1}
                  className={`relative inline-flex items-center px-2 py-2 rounded-l-md border border-gray-300 bg-white text-sm font-medium ${
                    page === 1 ? 'text-gray-300 cursor-not-allowed' : 'text-gray-500 hover:bg-gray-50'
                  }`}
                >
                  <span className="sr-only">Предыдущая</span>
                  <svg className="h-5 w-5" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                    <path fillRule="evenodd" d="M12.707 5.293a1 1 0 010 1.414L9.414 10l3.293 3.293a1 1 0 01-1.414 1.414l-4-4a1 1 0 010-1.414l4-4a1 1 0 011.414 0z" clipRule="evenodd" />
                  </svg>
                </button>
                
                {Array.from({ length: totalPages }).map((_, index) => (
                  <button
                    key={index}
                    onClick={() => goToPage(index + 1)}
                    className={`relative inline-flex items-center px-4 py-2 border text-sm font-medium ${
                      page === index + 1
                        ? 'z-10 bg-blue-50 border-blue-500 text-blue-600'
                        : 'bg-white border-gray-300 text-gray-500 hover:bg-gray-50'
                    }`}
                  >
                    {index + 1}
                  </button>
                ))}
                
                <button
                  onClick={() => goToPage(page + 1)}
                  disabled={page === totalPages}
                  className={`relative inline-flex items-center px-2 py-2 rounded-r-md border border-gray-300 bg-white text-sm font-medium ${
                    page === totalPages ? 'text-gray-300 cursor-not-allowed' : 'text-gray-500 hover:bg-gray-50'
                  }`}
                >
                  <span className="sr-only">Следующая</span>
                  <svg className="h-5 w-5" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                    <path fillRule="evenodd" d="M7.293 14.707a1 1 0 010-1.414L10.586 10 7.293 6.707a1 1 0 011.414-1.414l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414 0z" clipRule="evenodd" />
                  </svg>
                </button>
              </nav>
            </div>
          )}
          
          <div className="mt-8 flex justify-center">
            <Link href={`/quizzes/${id}`}>
              <Button variant="outline">Вернуться к викторине</Button>
            </Link>
          </div>
        </div>
      </div>
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

export default QuizResultsPage; 