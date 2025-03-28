import React, { useState } from 'react';
import { NextPage, GetServerSideProps } from 'next';
import { useRouter } from 'next/router';
import Layout from '../../components/layout/Layout';
import Button from '../../components/common/Button';
import { useQuizWithQuestions, useJoinQuiz, useLeaveQuiz } from '../../hooks/query-hooks/useQuizQueries';
import { formatDateTime, formatDuration } from '../../utils/date-utils';
import { useAppSelector } from '../../hooks/redux-hooks';
import { QuizStatus } from '../../types/quiz';
import Link from 'next/link';

const QuizDetailsPage: NextPage<{ id: string }> = ({ id }) => {
  const router = useRouter();
  const { isAuthenticated, user } = useAppSelector(state => state.auth);
  const [joinCode, setJoinCode] = useState<string>('');
  const [showJoinModal, setShowJoinModal] = useState<boolean>(false);
  
  // Получаем данные о викторине
  const { data: quiz, isLoading, isError, error } = useQuizWithQuestions(id);
  
  // Хуки для присоединения и выхода из викторины
  const joinQuizMutation = useJoinQuiz();
  const leaveQuizMutation = useLeaveQuiz();
  
  // Функция для определения, является ли пользователь участником викторины
  const isParticipant = () => {
    if (!quiz || !isAuthenticated || !user) return false;
    return quiz.participants?.some(participant => participant.user_id === user.id);
  };
  
  // Функция для определения, является ли пользователь автором викторины
  const isAuthor = () => {
    if (!quiz || !isAuthenticated || !user) return false;
    return quiz.user_id === user.id;
  };
  
  // Обработчик присоединения к викторине
  const handleJoinQuiz = () => {
    if (!isAuthenticated) {
      router.push(`/auth/login?redirect=${encodeURIComponent(router.asPath)}`);
      return;
    }
    
    joinQuizMutation.mutate({ 
      quizId: id, 
      joinCode: quiz?.requires_join_code ? joinCode : undefined 
    }, {
      onSuccess: () => {
        setShowJoinModal(false);
        setJoinCode('');
      }
    });
  };
  
  // Обработчик выхода из викторины
  const handleLeaveQuiz = () => {
    if (confirm('Вы уверены, что хотите покинуть эту викторину?')) {
      leaveQuizMutation.mutate({ quizId: id });
    }
  };
  
  // Обработчик для начала прохождения викторины
  const handleStartQuiz = () => {
    router.push(`/quizzes/${id}/play`);
  };
  
  // Функция для определения статуса викторины
  const renderStatusBadge = () => {
    if (!quiz) return null;
    
    const badgeClasses = {
      [QuizStatus.DRAFT]: 'bg-gray-100 text-gray-800',
      [QuizStatus.PUBLISHED]: 'bg-blue-100 text-blue-800',
      [QuizStatus.ACTIVE]: 'bg-green-100 text-green-800',
      [QuizStatus.COMPLETED]: 'bg-purple-100 text-purple-800',
      [QuizStatus.CANCELLED]: 'bg-red-100 text-red-800'
    };
    
    const statusLabels = {
      [QuizStatus.DRAFT]: 'Черновик',
      [QuizStatus.PUBLISHED]: 'Опубликовано',
      [QuizStatus.ACTIVE]: 'Активна',
      [QuizStatus.COMPLETED]: 'Завершена',
      [QuizStatus.CANCELLED]: 'Отменена'
    };
    
    return (
      <span className={`inline-flex items-center px-3 py-0.5 rounded-full text-sm font-medium ${badgeClasses[quiz.status]}`}>
        {statusLabels[quiz.status]}
      </span>
    );
  };
  
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
        <div className="bg-red-50 p-4 rounded-md mb-6">
          <div className="flex">
            <div className="flex-shrink-0">
              <svg className="h-5 w-5 text-red-400" viewBox="0 0 20 20" fill="currentColor">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
              </svg>
            </div>
            <div className="ml-3">
              <h3 className="text-sm font-medium text-red-800">
                Ошибка при загрузке викторины
              </h3>
              <div className="mt-2 text-sm text-red-700">
                <p>
                  {error instanceof Error ? error.message : 'Викторина не найдена или произошла ошибка при загрузке данных. Попробуйте обновить страницу.'}
                </p>
              </div>
            </div>
          </div>
        </div>
        <div className="mt-6">
          <Link href="/quizzes">
            <Button variant="outline">Вернуться к списку викторин</Button>
          </Link>
        </div>
      </Layout>
    );
  }
  
  return (
    <Layout title={quiz.title}>
      <div className="bg-white rounded-lg shadow-sm overflow-hidden">
        {/* Шапка викторины */}
        <div className="px-6 py-5 border-b border-gray-200 flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
          <div>
            <div className="flex items-center space-x-3">
              <h1 className="text-2xl font-bold text-gray-900">{quiz.title}</h1>
              {renderStatusBadge()}
            </div>
            
            <div className="mt-2 flex items-center text-sm text-gray-500">
              <span>Создано {formatDateTime(quiz.created_at)}</span>
              <span className="mx-2">•</span>
              <span>Категория: {quiz.category}</span>
              <span className="mx-2">•</span>
              <span>Сложность: {
                {
                  'easy': 'Легкая',
                  'medium': 'Средняя',
                  'hard': 'Сложная'
                }[quiz.difficulty] || quiz.difficulty
              }</span>
            </div>
          </div>
          
          <div className="flex space-x-3">
            {isAuthor() && (
              <Link href={`/quizzes/${id}/edit`}>
                <Button variant="outline">Редактировать</Button>
              </Link>
            )}
            
            {isParticipant() ? (
              <>
                {quiz.status === QuizStatus.ACTIVE && (
                  <Button 
                    variant="primary"
                    onClick={handleStartQuiz}
                  >
                    Пройти викторину
                  </Button>
                )}
                {quiz.status === QuizStatus.PUBLISHED && (
                  <Button 
                    variant="outline"
                    onClick={handleLeaveQuiz}
                    isLoading={leaveQuizMutation.isLoading}
                  >
                    Покинуть
                  </Button>
                )}
              </>
            ) : (
              quiz.status === QuizStatus.PUBLISHED && (
                <Button 
                  variant="primary"
                  onClick={() => quiz.requires_join_code ? setShowJoinModal(true) : handleJoinQuiz()}
                  isLoading={joinQuizMutation.isLoading}
                >
                  Присоединиться
                </Button>
              )
            )}
          </div>
        </div>
        
        {/* Информация о викторине */}
        <div className="px-6 py-5">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <div className="md:col-span-2">
              <h2 className="text-lg font-medium text-gray-900">Описание</h2>
              <div className="mt-2 text-gray-600 whitespace-pre-line">
                {quiz.description || 'Описание отсутствует'}
              </div>
              
              <div className="mt-6">
                <h2 className="text-lg font-medium text-gray-900">
                  Вопросы ({quiz.questions.length})
                </h2>
                <div className="mt-2 space-y-4">
                  {quiz.questions.map((question, index) => (
                    <div key={question.id} className="border rounded-md p-4">
                      <h3 className="font-medium text-gray-900">
                        Вопрос {index + 1}: {question.content}
                      </h3>
                      {(quiz.status === QuizStatus.COMPLETED || isAuthor()) && (
                        <div className="mt-2 grid grid-cols-1 sm:grid-cols-2 gap-2">
                          {question.answers.map(answer => (
                            <div 
                              key={answer.id} 
                              className={`p-2 rounded-md text-sm ${answer.is_correct 
                                ? 'bg-green-50 text-green-800 border border-green-200' 
                                : 'bg-gray-50 text-gray-800 border border-gray-200'
                              }`}
                            >
                              {answer.content}
                              {answer.is_correct && (
                                <span className="ml-2 text-green-600">✓ Правильный ответ</span>
                              )}
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            </div>
            
            <div className="bg-gray-50 rounded-lg p-5">
              <h2 className="text-lg font-medium text-gray-900">Информация</h2>
              <dl className="mt-4 space-y-4">
                <div>
                  <dt className="text-sm font-medium text-gray-500">Количество вопросов</dt>
                  <dd className="mt-1 text-gray-900">{quiz.questions.length}</dd>
                </div>
                
                <div>
                  <dt className="text-sm font-medium text-gray-500">Тайминг</dt>
                  <dd className="mt-1 text-gray-900">{formatDuration(quiz.time_limit)}</dd>
                </div>
                
                {quiz.start_time && (
                  <div>
                    <dt className="text-sm font-medium text-gray-500">Время начала</dt>
                    <dd className="mt-1 text-gray-900">{formatDateTime(quiz.start_time)}</dd>
                  </div>
                )}
                
                <div>
                  <dt className="text-sm font-medium text-gray-500">Участники</dt>
                  <dd className="mt-1 text-gray-900">{quiz.participants?.length || 0}</dd>
                </div>
                
                <div>
                  <dt className="text-sm font-medium text-gray-500">Автор</dt>
                  <dd className="mt-1 text-gray-900">{quiz.user?.username || 'Неизвестно'}</dd>
                </div>
                
                {quiz.requires_join_code && (
                  <div>
                    <dt className="text-sm font-medium text-gray-500">Требуется код для присоединения</dt>
                    <dd className="mt-1 text-gray-900">Да</dd>
                  </div>
                )}
              </dl>
              
              {quiz.status === QuizStatus.COMPLETED && (
                <div className="mt-6">
                  <Link href={`/quizzes/${id}/results`}>
                    <Button variant="outline" className="w-full">
                      Посмотреть результаты
                    </Button>
                  </Link>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
      
      {/* Модальное окно для ввода кода присоединения */}
      {showJoinModal && (
        <div className="fixed inset-0 overflow-y-auto z-50">
          <div className="flex items-center justify-center min-h-screen pt-4 px-4 pb-20 text-center sm:block sm:p-0">
            <div className="fixed inset-0 transition-opacity" aria-hidden="true">
              <div className="absolute inset-0 bg-gray-500 opacity-75"></div>
            </div>
            
            <span className="hidden sm:inline-block sm:align-middle sm:h-screen" aria-hidden="true">&#8203;</span>
            
            <div className="inline-block align-bottom bg-white rounded-lg text-left overflow-hidden shadow-xl transform transition-all sm:my-8 sm:align-middle sm:max-w-lg sm:w-full">
              <div className="bg-white px-4 pt-5 pb-4 sm:p-6 sm:pb-4">
                <div className="sm:flex sm:items-start">
                  <div className="mt-3 text-center sm:mt-0 sm:ml-4 sm:text-left w-full">
                    <h3 className="text-lg leading-6 font-medium text-gray-900" id="modal-title">
                      Введите код для присоединения
                    </h3>
                    <div className="mt-4">
                      <input
                        type="text"
                        value={joinCode}
                        onChange={(e) => setJoinCode(e.target.value)}
                        placeholder="Код для присоединения"
                        className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
                      />
                      {joinQuizMutation.isError && (
                        <p className="mt-2 text-sm text-red-600">
                          {joinQuizMutation.error instanceof Error 
                            ? joinQuizMutation.error.message 
                            : 'Ошибка при присоединении к викторине'}
                        </p>
                      )}
                    </div>
                  </div>
                </div>
              </div>
              <div className="bg-gray-50 px-4 py-3 sm:px-6 sm:flex sm:flex-row-reverse">
                <Button
                  variant="primary"
                  onClick={handleJoinQuiz}
                  className="w-full sm:w-auto sm:ml-3"
                  isLoading={joinQuizMutation.isLoading}
                >
                  Присоединиться
                </Button>
                <Button
                  variant="outline"
                  onClick={() => {
                    setShowJoinModal(false);
                    setJoinCode('');
                  }}
                  className="mt-3 w-full sm:mt-0 sm:w-auto"
                >
                  Отмена
                </Button>
              </div>
            </div>
          </div>
        </div>
      )}
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

export default QuizDetailsPage; 