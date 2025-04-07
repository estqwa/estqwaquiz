"use client";

import { useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { getQuizResults, QuizResult, getQuizById, Quiz } from '@/lib/api/quizzes'; // Импортируем getQuizById и Quiz
import { ApiError } from '@/lib/api/http-client';
import { useAuth } from '@/lib/auth/auth-context';
import Link from 'next/link';

export default function QuizResultsPage() {
  const params = useParams();
  const router = useRouter();
  const { user } = useAuth(); // Получаем текущего пользователя
  const quizId = parseInt(params.quizId as string, 10);

  const [results, setResults] = useState<QuizResult[]>([]);
  const [quizInfo, setQuizInfo] = useState<Quiz | null>(null); // Состояние для информации о викторине
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (isNaN(quizId)) {
      setError('Некорректный ID викторины');
      setLoading(false);
      return;
    }

    const fetchResults = async () => {
      try {
        setLoading(true);
        setError(null);

        // Загружаем и информацию о викторине, и результаты
        const [fetchedResults, fetchedQuizInfo] = await Promise.all([
          getQuizResults(quizId),
          getQuizById(quizId) // Загружаем детали викторины
        ]);

        setResults(fetchedResults);
        setQuizInfo(fetchedQuizInfo);

      } catch (err) {
        console.error('Ошибка загрузки результатов викторины:', err);
        setError((err as ApiError).error || 'Не удалось загрузить результаты');
      } finally {
        setLoading(false);
      }
    };

    fetchResults();
  }, [quizId]);

  const renderPodium = () => {
    const podium = results.slice(0, 3);
    const podiumStyles = [
      'bg-yellow-400 text-white border-yellow-600', // 1st
      'bg-gray-300 text-gray-700 border-gray-400', // 2nd
      'bg-orange-400 text-white border-orange-600' // 3rd
    ];

    return (
      <div className="flex justify-center items-end space-x-4 mb-12">
        {/* 2nd Place */}
        {podium.length >= 2 && (
          <div className="text-center">
            <div className={`w-24 h-32 rounded-t-lg p-4 border-b-4 flex flex-col justify-end items-center ${podiumStyles[1]}`}>
              <div className="font-bold text-2xl mb-1">2</div>
              <div className="text-sm truncate w-full">{podium[1].username}</div>
              <div className="text-xs font-semibold">{podium[1].score}</div>
            </div>
          </div>
        )}
        {/* 1st Place */}
        {podium.length >= 1 && (
          <div className="text-center">
            <div className={`w-28 h-40 rounded-t-lg p-4 border-b-4 flex flex-col justify-end items-center ${podiumStyles[0]}`}>
              <div className="font-bold text-3xl mb-1">1</div>
              <div className="text-sm truncate w-full">{podium[0].username}</div>
              <div className="text-lg font-semibold">{podium[0].score}</div>
            </div>
          </div>
        )}
        {/* 3rd Place */}
        {podium.length >= 3 && (
          <div className="text-center">
            <div className={`w-24 h-28 rounded-t-lg p-4 border-b-4 flex flex-col justify-end items-center ${podiumStyles[2]}`}>
              <div className="font-bold text-xl mb-1">3</div>
              <div className="text-sm truncate w-full">{podium[2].username}</div>
              <div className="text-xs font-semibold">{podium[2].score}</div>
            </div>
          </div>
        )}
      </div>
    );
  };

  return (
    <div className="container mx-auto px-4 py-8">
      <h1 className="text-3xl font-bold mb-2">
        Результаты викторины: {quizInfo ? `"${quizInfo.title}"` : 'Загрузка...'}
      </h1>
      <p className="text-gray-600 mb-8">
        {quizInfo ? `Завершилась: ${new Date(quizInfo.updated_at).toLocaleString()}` : ''}
      </p>

      {loading && (
        <div className="flex justify-center items-center h-64">
          <div className="animate-spin rounded-full h-16 w-16 border-t-4 border-b-4 border-blue-500"></div>
        </div>
      )}

      {error && (
        <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded relative" role="alert">
          <strong className="font-bold">Ошибка!</strong>
          <span className="block sm:inline"> {error}</span>
        </div>
      )}

      {!loading && !error && results.length > 0 && (
        <>
          {/* Podium */}
          {renderPodium()}

          {/* Full Results Table */}
          <div className="bg-white shadow-md rounded-lg overflow-hidden">
            <table className="min-w-full leading-normal">
              <thead>
                <tr className="bg-gray-100">
                  <th className="px-5 py-3 border-b-2 border-gray-200 text-left text-xs font-semibold text-gray-600 uppercase tracking-wider">
                    Место
                  </th>
                  <th className="px-5 py-3 border-b-2 border-gray-200 text-left text-xs font-semibold text-gray-600 uppercase tracking-wider">
                    Игрок
                  </th>
                  <th className="px-5 py-3 border-b-2 border-gray-200 text-left text-xs font-semibold text-gray-600 uppercase tracking-wider">
                    Очки
                  </th>
                  <th className="px-5 py-3 border-b-2 border-gray-200 text-left text-xs font-semibold text-gray-600 uppercase tracking-wider">
                    Правильно
                  </th>
                  <th className="px-5 py-3 border-b-2 border-gray-200 text-left text-xs font-semibold text-gray-600 uppercase tracking-wider">
                    Приз
                  </th>
                </tr>
              </thead>
              <tbody>
                {results.map((result, index) => (
                  <tr key={result.id} className={`${result.user_id === user?.id ? 'bg-blue-50 font-medium' : 'hover:bg-gray-50'} ${result.is_eliminated ? 'opacity-60' : ''}`}>
                    <td className="px-5 py-4 border-b border-gray-200 bg-white text-sm">
                      <p className="text-gray-900 whitespace-no-wrap font-semibold">{result.rank || index + 1}</p>
                    </td>
                    <td className="px-5 py-4 border-b border-gray-200 bg-white text-sm">
                      <div className="flex items-center">
                        {/* TODO: Добавить аватарки, если они будут */} 
                        {/* <div className="flex-shrink-0 w-10 h-10">
                          <img className="w-full h-full rounded-full" src={result.profile_picture || '/default-avatar.png'} alt={result.username} />
                        </div> */}
                        <div className="ml-3">
                          <p className="text-gray-900 whitespace-no-wrap">
                            {result.username}
                            {result.user_id === user?.id && <span className="text-blue-600"> (Вы)</span>}
                          </p>
                          {result.is_eliminated && <span className="text-xs text-red-600">Выбыл</span>}
                        </div>
                      </div>
                    </td>
                    <td className="px-5 py-4 border-b border-gray-200 bg-white text-sm">
                      <p className="text-gray-900 whitespace-no-wrap font-bold">{result.score}</p>
                    </td>
                    <td className="px-5 py-4 border-b border-gray-200 bg-white text-sm">
                      <p className="text-gray-900 whitespace-no-wrap">
                        {result.correct_answers} / {result.total_questions}
                      </p>
                    </td>
                    <td className="px-5 py-4 border-b border-gray-200 bg-white text-sm">
                      {result.is_winner ? (
                        <span className="relative inline-block px-3 py-1 font-semibold text-green-900 leading-tight">
                          <span aria-hidden className="absolute inset-0 bg-green-200 opacity-50 rounded-full"></span>
                          <span className="relative">🏆 {result.prize_fund > 0 ? `${result.prize_fund} $` : 'Победитель!'}</span>
                        </span>
                      ) : (
                        <span className="text-gray-500">-</span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <div className="mt-8 text-center">
            <Link href="/quizzes" className="bg-blue-500 hover:bg-blue-700 text-white font-bold py-2 px-4 rounded">
              К списку викторин
            </Link>
          </div>
        </>
      )}

      {!loading && !error && results.length === 0 && (
        <div className="text-center text-gray-500 mt-12">
          <p>Результаты для этой викторины еще не опубликованы или недоступны.</p>
          <div className="mt-8">
            <Link href="/quizzes" className="bg-blue-500 hover:bg-blue-700 text-white font-bold py-2 px-4 rounded">
              К списку викторин
            </Link>
          </div>
        </div>
      )}
    </div>
  );
} 