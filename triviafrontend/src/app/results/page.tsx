"use client";

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { useAuth } from '../../lib/auth/auth-context';
import { Result } from '../../lib/api/results';
import { ApiError } from '../../lib/api/http-client';

export default function ResultsPage() {
  const { isAuthenticated, user } = useAuth();
  const [results, setResults] = useState<Result[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const loadResults = async () => {
      if (!isAuthenticated || !user) {
        setLoading(false);
        return;
      }
      
      try {
        setLoading(true);
        setError(null);
        
        // Получаем результаты пользователя
        // const userResults = await getUserResults();
        // setResults(userResults);
        
        // Устанавливаем ошибку, т.к. функционал не реализован на бэкенде
        setError("Функционал просмотра истории всех результатов временно недоступен."); 
        setResults([]); // Очищаем результаты

      } catch (err) {
        console.error('Ошибка загрузки результатов:', err);
        setError((err as ApiError)?.error || 'Ошибка при загрузке результатов');
      } finally {
        setLoading(false);
      }
    };

    loadResults();
  }, [isAuthenticated, user]);

  // Форматирование даты
  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    return new Intl.DateTimeFormat('ru-RU', {
      day: '2-digit',
      month: '2-digit', 
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    }).format(date);
  };

  return (
    <div>
      <h1 className="text-3xl font-bold mb-6">Мои результаты</h1>
      
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
      
      {!loading && results.length === 0 && (
        <div className="bg-white p-6 rounded-lg shadow-md">
          <p className="text-gray-600 mb-4">У вас пока нет результатов участия в викторинах.</p>
          <p>
            <Link href="/quizzes" className="text-blue-600 hover:text-blue-800">
              Перейти к списку доступных викторин
            </Link>
          </p>
        </div>
      )}
      
      {!loading && results.length > 0 && (
        <div className="bg-white rounded-lg shadow-md overflow-hidden">
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Викторина
                  </th>
                  <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Дата участия
                  </th>
                  <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Результат
                  </th>
                  <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Место
                  </th>
                  <th scope="col" className="px-6 py-3 text-center text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Действия
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {results.map((result) => (
                  <tr key={result.id} className="hover:bg-gray-50">
                    <td className="px-6 py-4 whitespace-nowrap">
                      <Link 
                        href={`/quizzes/${result.quiz_id}`}
                        className="text-blue-600 hover:text-blue-800 font-medium"
                      >
                        Викторина #{result.quiz_id}
                      </Link>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {formatDate(result.completed_at)}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="text-sm font-medium text-gray-900">
                        {result.score} очков
                      </div>
                      <div className="text-sm text-gray-500">
                        {result.correct_answers} из {result.total_questions} правильных
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      {result.rank === 1 && (
                        <span className="px-2 py-1 inline-flex text-xs leading-5 font-semibold rounded-full bg-yellow-100 text-yellow-800">
                          🥇 1 место
                        </span>
                      )}
                      {result.rank === 2 && (
                        <span className="px-2 py-1 inline-flex text-xs leading-5 font-semibold rounded-full bg-gray-100 text-gray-800">
                          🥈 2 место
                        </span>
                      )}
                      {result.rank === 3 && (
                        <span className="px-2 py-1 inline-flex text-xs leading-5 font-semibold rounded-full bg-yellow-50 text-yellow-700">
                          🥉 3 место
                        </span>
                      )}
                      {result.rank > 3 && (
                        <span className="px-2 py-1 inline-flex text-xs leading-5 font-semibold rounded-full bg-gray-50 text-gray-600">
                          {result.rank} место
                        </span>
                      )}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-center">
                      <Link 
                        href={`/results/${result.quiz_id}`}
                        className="text-indigo-600 hover:text-indigo-900"
                      >
                        Подробности
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
} 