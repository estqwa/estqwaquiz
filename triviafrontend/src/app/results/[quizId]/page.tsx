"use client";

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { useRouter, useParams } from 'next/navigation';
import { getQuizById, Quiz } from '../../../lib/api/quizzes';
import { getQuizResults, Result } from '../../../lib/api/results';
import { ApiError } from '../../../lib/api/http-client';
import { useAuth } from '../../../lib/auth/auth-context';
import { formatDate, DateFormat } from '../../../lib/utils/dateUtils';

export default function QuizResultPage() {
  const router = useRouter();
  const params = useParams();
  const { user, isAuthenticated } = useAuth();
  const [quizInfo, setQuizInfo] = useState<Quiz | null>(null);
  const [results, setResults] = useState<Result[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    // Функция загрузки результатов викторины
    const loadQuizResults = async () => {
      try {
        setLoading(true);
        setError(null);
        
        const quizId = parseInt(params.quizId as string, 10);
        if (isNaN(quizId)) {
          setError('Некорректный ID викторины');
          return;
        }
        
        // Загружаем информацию о викторине
        const quizData = await getQuizById(quizId);
        setQuizInfo(quizData);
        
        // Загружаем результаты викторины для таблицы лидеров
        const resultsData = await getQuizResults(quizId);
        setResults(resultsData);
      } catch (err) {
        console.error('Ошибка загрузки результатов викторины:', err);
        setError((err as ApiError).error || 'Ошибка при загрузке результатов викторины');
      } finally {
        setLoading(false);
      }
    };

    loadQuizResults();
  }, [params.quizId, isAuthenticated, user]);

  // Обработчик нажатия кнопки "Назад"
  const handleBack = () => {
    router.back();
  };

  // Вычисление процента правильных ответов
  const calculatePercentage = (correct: number, total: number): number => {
    if (total === 0) return 0;
    return Math.round((correct / total) * 100);
  };

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
      
      <h1 className="text-3xl font-bold mb-6">
        {loading ? 'Загрузка...' : quizInfo ? `Результаты: ${quizInfo.title}` : 'Результаты викторины'}
      </h1>
      
      {loading && (
        <div className="flex justify-center my-12">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
        </div>
      )}
      
      {error && (
        <div className="bg-red-100 p-4 rounded-lg text-red-800">
          <p>{error}</p>
        </div>
      )}
      
      {!loading && quizInfo && (
        <div className="bg-white rounded-lg shadow-md overflow-hidden">
          <div className="p-6">
            <h2 className="text-2xl font-bold mb-2">{quizInfo.title}</h2>
            <p className="text-gray-600 mb-6">
              Запланировано: {formatDate(quizInfo.scheduled_time, DateFormat.MEDIUM)}
            </p>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-8">
              <div className="bg-blue-50 p-4 rounded-md text-center">
                <h3 className="text-gray-600 text-sm mb-1">Статус викторины</h3>
                <p className="text-2xl font-bold text-blue-700">
                  {quizInfo.status === 'completed' ? 'Завершена' : 
                   quizInfo.status === 'in_progress' ? 'В процессе' : 
                   quizInfo.status === 'scheduled' ? 'Запланирована' : 'Неизвестно'}
                </p>
              </div>
              
              <div className="bg-purple-50 p-4 rounded-md text-center">
                <h3 className="text-gray-600 text-sm mb-1">Количество вопросов</h3>
                <p className="text-2xl font-bold text-purple-700">{quizInfo.question_count || 'Н/Д'}</p>
              </div>
            </div>
            
            <div className="flex justify-center">
              <button 
                onClick={() => router.push('/quizzes')}
                className="bg-blue-600 hover:bg-blue-700 text-white py-2 px-6 rounded-md transition-colors"
              >
                К списку викторин
              </button>
            </div>
          </div>
        </div>
      )}
      
      {!loading && results.length > 0 ? (
        <div className="bg-white p-8 rounded-lg shadow-md mt-6">
          <h2 className="text-2xl font-bold mb-6">Таблица лидеров</h2>
          <div className="overflow-x-auto">
            <table className="min-w-full bg-white">
              <thead>
                <tr className="bg-gray-100 text-gray-700">
                  <th className="py-3 px-4 text-left">Место</th>
                  <th className="py-3 px-4 text-left">Участник</th>
                  <th className="py-3 px-4 text-left">Очки</th>
                  <th className="py-3 px-4 text-left">Правильные ответы</th>
                </tr>
              </thead>
              <tbody>
                {results.map((result) => (
                  <tr 
                    key={result.id} 
                    className={`hover:bg-gray-50 ${user && result.user_id === user.id ? 'bg-blue-50' : ''}`}
                  >
                    <td className="py-3 px-4 border-b">
                      <div className="flex items-center">
                        {result.rank === 1 && <span className="text-yellow-500 mr-2">🥇</span>}
                        {result.rank === 2 && <span className="text-gray-400 mr-2">🥈</span>}
                        {result.rank === 3 && <span className="text-amber-600 mr-2">🥉</span>}
                        {result.rank}
                      </div>
                    </td>
                    <td className="py-3 px-4 border-b font-medium">
                      {result.username}
                      {user && result.user_id === user.id && ' (Вы)'}
                    </td>
                    <td className="py-3 px-4 border-b">{result.score}</td>
                    <td className="py-3 px-4 border-b">{result.correct_answers} / {result.total_questions}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      ) : !loading && (
        <div className="bg-white p-8 rounded-lg shadow-md text-center mt-6">
          <p className="text-gray-600">Результаты для этой викторины пока отсутствуют.</p>
          
          {quizInfo && quizInfo.status !== 'completed' && (
            <div className="mt-4">
              <p className="text-gray-700 mb-2">Викторина еще не завершена.</p>
              {quizInfo.status === 'in_progress' && (
                <Link href={`/quiz/${params.quizId as string}/live`} className="text-blue-600 hover:text-blue-800">
                  Перейти к участию →
                </Link>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
} 