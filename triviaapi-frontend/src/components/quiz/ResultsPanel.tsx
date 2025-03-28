import React, { memo, useMemo } from 'react';
import { UserQuizResult } from '../../types/result';
import { formatDateTime } from '../../utils/date-utils';
import Link from 'next/link';
import Button from '../common/Button';

interface ResultsPanelProps {
  quizId: number;
  quizTitle: string;
  results?: UserQuizResult | null;
  leaderboard?: UserQuizResult[] | null;
  limit?: number;
  showAllResultsButton?: boolean;
  isLoading?: boolean;
}

/**
 * Компонент для отображения результатов викторины и таблицы лидеров.
 * Оптимизирован с помощью React.memo для предотвращения лишних ререндеров.
 * 
 * @param {ResultsPanelProps} props - Свойства компонента
 * @param {number} props.quizId - ID викторины
 * @param {string} props.quizTitle - Название викторины
 * @param {UserQuizResult | null} [props.results] - Результаты текущего пользователя
 * @param {UserQuizResult[] | null} [props.leaderboard] - Таблица лидеров
 * @param {number} [props.limit=10] - Ограничение количества отображаемых результатов
 * @param {boolean} [props.showAllResultsButton=true] - Показывать ли кнопку "Посмотреть все результаты"
 * @param {boolean} [props.isLoading=false] - Состояние загрузки
 * 
 * @returns {React.ReactElement} Компонент для отображения результатов
 */
const ResultsPanel: React.FC<ResultsPanelProps> = ({
  quizId,
  quizTitle,
  results,
  leaderboard,
  limit = 10,
  showAllResultsButton = true,
  isLoading = false
}) => {
  // Мемоизируем вычисление отображаемых результатов для предотвращения лишних пересчетов
  const displayedLeaderboard = useMemo(() => {
    if (!leaderboard) return [];
    
    return limit > 0 
      ? leaderboard.slice(0, limit) 
      : leaderboard;
  }, [leaderboard, limit]);
  
  if (isLoading) {
    return (
      <div className="bg-white rounded-lg shadow-sm p-6">
        <div className="flex justify-center items-center h-40">
          <div className="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-blue-500"></div>
        </div>
      </div>
    );
  }
  
  return (
    <div className="bg-white rounded-lg shadow-sm overflow-hidden">
      <div className="px-6 py-5 border-b border-gray-200">
        <h2 className="text-xl font-semibold text-gray-900">
          Результаты викторины: {quizTitle}
        </h2>
      </div>
      
      {/* Результаты текущего пользователя */}
      {results && (
        <div className="px-6 py-4 bg-blue-50 border-b border-blue-100">
          <h3 className="text-lg font-medium text-gray-900 mb-3">Ваш результат</h3>
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
            <div className="bg-white p-4 rounded-lg shadow-sm">
              <p className="text-sm font-medium text-gray-500">Место</p>
              <p className="mt-1 text-2xl font-semibold text-gray-900">
                {results.rank || '-'}
              </p>
            </div>
            <div className="bg-white p-4 rounded-lg shadow-sm">
              <p className="text-sm font-medium text-gray-500">Баллы</p>
              <p className="mt-1 text-2xl font-semibold text-gray-900">
                {results.score} / {results.total_questions * 100}
              </p>
            </div>
            <div className="bg-white p-4 rounded-lg shadow-sm">
              <p className="text-sm font-medium text-gray-500">Правильные ответы</p>
              <p className="mt-1 text-2xl font-semibold text-gray-900">
                {results.correct_answers} / {results.total_questions}
              </p>
            </div>
          </div>
        </div>
      )}
      
      {/* Таблица лидеров */}
      <div className="px-6 py-5">
        <h3 className="text-lg font-medium text-gray-900 mb-4">Таблица лидеров</h3>
        
        {(!leaderboard || leaderboard.length === 0) ? (
          <div className="text-center py-8">
            <p className="text-gray-500">Нет доступных результатов</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Место
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
                    Завершено
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {displayedLeaderboard.map((item, index) => (
                  <tr 
                    key={item.id || `result-${index}`} 
                    className={results && item.user_id === results.user_id ? 'bg-blue-50' : ''}
                  >
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="text-sm font-medium text-gray-900">
                        {item.rank || index + 1}
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="flex items-center">
                        <div className="h-8 w-8 rounded-full bg-gray-200 flex items-center justify-center">
                          <span className="text-sm font-medium text-gray-600">
                            {item.username?.charAt(0).toUpperCase() || '?'}
                          </span>
                        </div>
                        <div className="ml-4">
                          <div className="text-sm font-medium text-gray-900">
                            {item.username || 'Неизвестный участник'}
                          </div>
                        </div>
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="text-sm text-gray-900">
                        {item.score}
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="text-sm text-gray-900">
                        {item.correct_answers} / {item.total_questions || '-'}
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {formatDateTime(item.completed_at)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
        
        {/* Кнопка для просмотра всех результатов */}
        {showAllResultsButton && leaderboard && leaderboard.length > 0 && (
          <div className="mt-6 flex justify-center">
            <Link href={`/quizzes/${quizId}/results`}>
              <Button variant="outline">
                {leaderboard.length > limit ? 'Посмотреть все результаты' : 'Подробные результаты'}
              </Button>
            </Link>
          </div>
        )}
      </div>
    </div>
  );
};

// Оборачиваем компонент в React.memo для предотвращения лишних перерисовок
export default memo(ResultsPanel); 