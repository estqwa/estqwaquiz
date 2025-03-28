import React from 'react';
import Link from 'next/link';
import { Quiz } from '../../types/quiz';
import { formatDate, formatDateTime } from '../../utils/dateFormat';

interface QuizCardProps {
  quiz: Quiz;
}

export const QuizCard: React.FC<QuizCardProps> = ({ quiz }) => {
  const getStatusBadge = () => {
    switch (quiz.status) {
      case 'draft':
        return <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800">Черновик</span>;
      case 'published':
        return <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800">Опубликована</span>;
      case 'active':
        return <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">Активна</span>;
      case 'completed':
        return <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-purple-100 text-purple-800">Завершена</span>;
      case 'cancelled':
        return <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800">Отменена</span>;
      default:
        return null;
    }
  };

  return (
    <div className="border border-gray-200 rounded-lg overflow-hidden shadow-sm hover:shadow-md transition-shadow bg-white">
      <div className="p-5">
        <div className="flex justify-between items-start">
          <h3 className="text-lg font-semibold text-gray-900 mb-1">{quiz.title}</h3>
          {getStatusBadge()}
        </div>
        
        <p className="text-sm text-gray-600 mb-3 line-clamp-2">{quiz.description}</p>
        
        <div className="mb-4 flex flex-wrap gap-2">
          <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800">
            {quiz.category}
          </span>
          <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
            {quiz.difficulty}
          </span>
          <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-purple-100 text-purple-800">
            {quiz.question_count} вопросов
          </span>
        </div>
        
        <div className="grid grid-cols-2 gap-2 text-xs text-gray-500 mb-4">
          <div>
            <span className="block font-medium">Начало:</span>
            <span>{formatDateTime(quiz.start_time)}</span>
          </div>
          <div>
            <span className="block font-medium">Длительность:</span>
            <span>{quiz.duration_minutes} минут</span>
          </div>
          <div>
            <span className="block font-medium">Создано:</span>
            <span>{formatDate(quiz.created_at)}</span>
          </div>
          <div>
            <span className="block font-medium">Обновлено:</span>
            <span>{formatDate(quiz.updated_at)}</span>
          </div>
        </div>
        
        <div className="flex justify-between items-center">
          <Link href={`/quiz/${quiz.id}`} className="inline-flex items-center text-sm font-medium text-blue-600 hover:text-blue-500">
            Подробнее
            <svg className="ml-1 w-4 h-4" fill="currentColor" viewBox="0 0 20 20" xmlns="http://www.w3.org/2000/svg">
              <path fillRule="evenodd" d="M7.293 14.707a1 1 0 010-1.414L10.586 10 7.293 6.707a1 1 0 011.414-1.414l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414 0z" clipRule="evenodd"></path>
            </svg>
          </Link>
          
          {quiz.status === 'published' && (
            <Link href={`/quiz/${quiz.id}/join`} className="inline-flex items-center px-3 py-1.5 border border-transparent text-xs font-medium rounded-md shadow-sm text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500">
              Присоединиться
            </Link>
          )}
        </div>
      </div>
    </div>
  );
};

export default QuizCard; 