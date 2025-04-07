"use client";

import Link from 'next/link';
import { useEffect, useState } from 'react';
import { useAuth } from '../../lib/auth/auth-context';
import { getAvailableQuizzes, Quiz } from '../../lib/api/quizzes';
import { ApiError } from '../../lib/api/http-client';
import { formatDate, DateFormat } from '../../lib/utils/dateUtils';

export default function QuizzesPage() {
  const [quizzes, setQuizzes] = useState<Quiz[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const { isAuthenticated } = useAuth();

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

  // Получение цвета для статуса
  const getStatusColor = (status: string): string => {
    switch (status) {
      case 'scheduled':
        return 'text-blue-600';
      case 'in_progress':
        return 'text-green-600';
      case 'completed':
        return 'text-gray-600';
      default:
        return 'text-gray-700';
    }
  };

  useEffect(() => {
    const fetchQuizzes = async () => {
      try {
        setLoading(true);
        const data = await getAvailableQuizzes();
        setQuizzes(data);
        setError(null);
      } catch (err) {
        console.error('Ошибка загрузки викторин:', err);
        setError((err as ApiError).error || 'Не удалось загрузить список викторин');
      } finally {
        setLoading(false);
      }
    };

    fetchQuizzes();
  }, []);

  return (
    <div className="container mx-auto px-4 py-8">
      <h1 className="text-3xl font-bold mb-6">Доступные викторины</h1>
      
      {loading ? (
        <div className="flex justify-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
        </div>
      ) : error ? (
        <div className="bg-red-100 p-4 rounded-lg text-red-800">
          <p>{error}</p>
        </div>
      ) : quizzes.length === 0 ? (
        <div className="text-center py-8">
          <p className="text-xl text-gray-600">Нет доступных викторин</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {quizzes.map(quiz => (
            <div key={quiz.id} className="bg-white p-6 rounded-lg shadow-md transition-shadow hover:shadow-lg">
              <h2 className="text-xl font-bold mb-2">{quiz.title}</h2>
              <p className="text-gray-600 mb-4">{quiz.description}</p>
              
              <div className="flex justify-between items-center mb-3">
                <span className={`font-medium ${getStatusColor(quiz.status)}`}>
                  {getStatusText(quiz.status)}
                </span>
                <span className="text-gray-500 text-sm">
                  {quiz.question_count} вопросов
                </span>
              </div>
              
              <p className="text-sm text-gray-500 mb-4">
                Дата: {formatDate(quiz.scheduled_time, DateFormat.SHORT)}
              </p>
              
              <Link 
                href={`/quizzes/${quiz.id}`}
                className="inline-block bg-blue-600 hover:bg-blue-700 text-white py-2 px-4 rounded transition-colors w-full text-center"
              >
                Подробнее
              </Link>
            </div>
          ))}
        </div>
      )}
    </div>
  );
} 