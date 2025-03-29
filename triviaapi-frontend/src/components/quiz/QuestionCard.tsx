import React, { useState, useEffect, useCallback, memo } from 'react';
import { Question } from '../../types/question';

interface QuestionCardProps {
  question: Question;
  remainingTime?: number | null;
  isSubmitting?: boolean;
  disabled?: boolean;
  hasSubmitted?: boolean;
  correctOptionId?: number | null;
  onAnswer?: (answerId: string) => void;
  onSkip?: () => void;
}

/**
 * Компонент для отображения вопроса викторины и вариантов ответа
 * Оптимизирован с помощью React.memo для предотвращения лишних ререндеров
 */
const QuestionCard: React.FC<QuestionCardProps> = ({
  question,
  remainingTime,
  isSubmitting = false,
  disabled = false,
  hasSubmitted = false,
  correctOptionId = null,
  onAnswer,
  onSkip
}) => {
  const [selectedOption, setSelectedOption] = useState<string | null>(null);
  const [timeLeft, setTimeLeft] = useState<number | null>(null);
  
  // Установка начального времени при изменении вопроса или времени
  useEffect(() => {
    console.log('QuestionCard получил remainingTime:', remainingTime);
    if (remainingTime !== undefined && remainingTime !== null) {
      setTimeLeft(remainingTime);
      console.log('QuestionCard установил timeLeft:', remainingTime);
    } else if (question.timeLimitSeconds) {
      setTimeLeft(question.timeLimitSeconds);
      console.log('QuestionCard установил timeLeft из timeLimitSeconds:', question.timeLimitSeconds);
    }
  }, [question, remainingTime]);
  
  // Отображаем таймер даже если он не связан с вопросом (например, обратный отсчет до начала)
  useEffect(() => {
    console.log('QuestionCard компонент отрисован с timeLeft:', timeLeft);
  }, [timeLeft]);
  
  // Мемоизированный обработчик выбора ответа
  const handleOptionSelect = useCallback((optionId: string) => {
    if (disabled || hasSubmitted) return;
    setSelectedOption(optionId);
  }, [disabled, hasSubmitted]);
  
  // Мемоизированный обработчик отправки ответа
  const handleSubmit = useCallback(() => {
    if (!selectedOption || disabled || hasSubmitted) return;
    if (onAnswer) {
      onAnswer(selectedOption);
    }
  }, [selectedOption, disabled, hasSubmitted, onAnswer]);
  
  // Мемоизированный обработчик пропуска вопроса
  const handleSkip = useCallback(() => {
    if (onSkip) {
      onSkip();
    }
  }, [onSkip]);
  
  // Мемоизированная функция форматирования времени
  const formatTime = useCallback((seconds: number): string => {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}:${secs < 10 ? '0' : ''}${secs}`;
  }, []);
  
  // Мемоизированная функция определения класса для таймера
  const getTimerClass = useCallback((): string => {
    if (!timeLeft) return 'text-gray-700';
    
    if (timeLeft <= 10) {
      return 'text-red-600 animate-pulse font-bold';
    } else if (timeLeft <= 20) {
      return 'text-orange-500 font-bold';
    } else {
      return 'text-green-600';
    }
  }, [timeLeft]);
  
  // Мемоизированная функция определения класса для варианта ответа
  const getOptionClass = useCallback((optionId: string): string => {
    if (hasSubmitted && correctOptionId !== null) {
      if (optionId === correctOptionId.toString()) {
        return 'border-green-500 bg-green-50 text-green-800';
      } else if (optionId === selectedOption) {
        return 'border-red-500 bg-red-50 text-red-800';
      }
    }
    
    return selectedOption === optionId 
      ? 'border-blue-500 bg-blue-50 ring-2 ring-blue-200' 
      : 'border-gray-300 hover:bg-gray-50';
  }, [hasSubmitted, correctOptionId, selectedOption]);
  
  return (
    <div className="bg-white rounded-lg shadow-sm overflow-hidden">
      <div className="px-6 py-5 border-b border-gray-200 flex justify-between items-center">
        <h2 className="text-xl font-semibold text-gray-900">
          Вопрос {question.questionNumber || '#'}
        </h2>
        
        {timeLeft !== null && (
          <div className={`text-lg font-medium ${getTimerClass()}`}>
            {formatTime(timeLeft)}
          </div>
        )}
      </div>
      
      <div className="px-6 py-6">
        <div className="mb-6">
          <p className="text-lg text-gray-900 font-medium">
            {question.text}
          </p>
        </div>
        
        <div className="space-y-3 mb-8">
          {question.options.map((option) => (
            <div
              key={option.id}
              onClick={() => handleOptionSelect(option.id.toString())}
              className={`p-4 border rounded-lg cursor-pointer transition-colors ${getOptionClass(option.id.toString())} ${
                disabled ? 'opacity-70 cursor-not-allowed' : ''
              }`}
            >
              <p className="text-gray-900">{option.text}</p>
              {hasSubmitted && correctOptionId !== null && option.id === correctOptionId && (
                <div className="mt-2 text-sm text-green-600 flex items-center">
                  <svg className="h-5 w-5 mr-1" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
                    <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                  </svg>
                  Правильный ответ
                </div>
              )}
            </div>
          ))}
        </div>
        
        <div className="flex justify-between">
          {onSkip && (
            <button
              type="button"
              onClick={handleSkip}
              disabled={disabled || isSubmitting}
              className="px-4 py-2 border border-gray-300 rounded-md text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Пропустить
            </button>
          )}
          
          <button
            type="button"
            onClick={handleSubmit}
            disabled={!selectedOption || disabled || isSubmitting || hasSubmitted}
            className="px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isSubmitting ? (
              <span className="flex items-center">
                <svg className="animate-spin -ml-1 mr-2 h-4 w-4 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
                Отправка...
              </span>
            ) : hasSubmitted ? 'Ответ отправлен' : 'Ответить'}
          </button>
        </div>
      </div>
    </div>
  );
};

// Оборачиваем компонент в React.memo для предотвращения лишних перерисовок
export default memo(QuestionCard); 