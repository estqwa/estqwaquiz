"use client";

import { useEffect, useState } from 'react';
import { useRouter, useParams } from 'next/navigation';
import { getQuizById, Quiz } from '../../../lib/api/quizzes';
import { ApiError } from '../../../lib/api/http-client';
import { formatDate, DateFormat } from '../../../lib/utils/dateUtils';

export default function QuizDetailsPage() {
  const router = useRouter();
  const params = useParams();
  const [quiz, setQuiz] = useState<Quiz | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Получение статуса на русском языке
  const getStatusText = (status: string): string => {
    switch (status) {
      case 'scheduled':
        return 'Запланирована';
      case 'in_progress':
        return 'В процессе';
      case 'completed':
        return 'Завершена';
      default:
        return status;
    }
  };

  useEffect(() => {
    const fetchQuizDetails = async () => {
      try {
        setLoading(true);
        
        const quizId = parseInt(params.quizId as string, 10);
        if (isNaN(quizId)) {
          setError('Некорректный ID викторины');
          return;
        }
        
        const quizData = await getQuizById(quizId);
        setQuiz(quizData);
      } catch (err) {
        console.error('Ошибка загрузки викторины:', err);
        setError((err as ApiError).error || 'Ошибка при загрузке информации о викторине');
      } finally {
        setLoading(false);
      }
    };

    fetchQuizDetails();
  }, [params.quizId]);

  // Обработчик нажатия кнопки "Назад"
  const handleBack = () => {
    router.back();
  };

  // Отображение детальной информации о викторине
  return (
    <div className="container mx-auto px-4 py-8">
      <button 
        onClick={handleBack}
        className="mb-6 flex items-center text-blue-600 hover:text-blue-800"
      >
        <svg className="w-5 h-5 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
        </svg>
        Назад
      </button>

      {loading ? (
        <div className="flex justify-center my-12">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
        </div>
      ) : error ? (
        <div className="bg-red-100 p-4 rounded-lg text-red-800">
          <p>{error}</p>
        </div>
      ) : quiz ? (
        <div className="bg-white rounded-lg shadow-md p-6">
          <h1 className="text-3xl font-bold mb-4">{quiz.title}</h1>
          
          <div className="mb-6">
            <p className="text-gray-700 text-lg mb-4">{quiz.description}</p>
            
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
              <div className="bg-gray-50 p-4 rounded-md">
                <h3 className="text-gray-500 text-sm mb-1">Статус</h3>
                <p className="text-lg font-medium">{getStatusText(quiz.status)}</p>
              </div>
              
              <div className="bg-gray-50 p-4 rounded-md">
                <h3 className="text-gray-500 text-sm mb-1">Дата проведения</h3>
                <p className="text-lg font-medium">{formatDate(quiz.scheduled_time, DateFormat.MEDIUM)}</p>
              </div>
              
              <div className="bg-gray-50 p-4 rounded-md">
                <h3 className="text-gray-500 text-sm mb-1">Количество вопросов</h3>
                <p className="text-lg font-medium">{quiz.question_count}</p>
              </div>
              
              <div className="bg-gray-50 p-4 rounded-md">
                <h3 className="text-gray-500 text-sm mb-1">Время до начала</h3>
                <p className="text-lg font-medium">{formatDate(quiz.scheduled_time, DateFormat.RELATIVE)}</p>
              </div>
            </div>
          </div>
          
          <div className="flex flex-col sm:flex-row gap-4">
            {quiz.status === 'scheduled' && (
              <button 
                onClick={() => router.push(`/quiz/${quiz.id}/live`)}
                className="bg-blue-600 hover:bg-blue-700 text-white py-2 px-6 rounded-md transition-colors"
              >
                Перейти к викторине
              </button>
            )}
            
            {quiz.status === 'in_progress' && (
              <button 
                onClick={() => router.push(`/quiz/${quiz.id}/live`)}
                className="bg-green-600 hover:bg-green-700 text-white py-2 px-6 rounded-md transition-colors"
              >
                Присоединиться к викторине
              </button>
            )}
            
            {quiz.status === 'completed' && (
              <button 
                onClick={() => router.push(`/results/${quiz.id}`)}
                className="bg-green-600 hover:bg-green-700 text-white py-2 px-6 rounded-md transition-colors"
              >
                Посмотреть результаты
              </button>
            )}
          </div>
        </div>
      ) : (
        <div className="text-center py-8">
          <p className="text-xl text-gray-600">Викторина не найдена</p>
        </div>
      )}
    </div>
  );
} 