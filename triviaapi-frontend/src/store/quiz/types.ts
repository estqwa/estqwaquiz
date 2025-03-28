// Типы для Quiz Slice
import { Quiz } from '../../types/quiz';
import { Question } from '../../types/question';
import { UserAnswer } from '../../types/answer';
import { UserQuizResult } from '../../types/result';

/**
 * Состояние викторины для Redux-слайса
 */
export interface QuizState {
  activeQuiz: Quiz | null; // Информация о текущей активной викторине
  currentQuestion: Question | null; // Текущий вопрос (полученный по WS)
  questions: Question[]; // Список всех вопросов викторины (если они доступны сразу)
  userAnswers: Record<number, UserAnswer>; // Ответы пользователя в текущей викторине (questionId -> UserAnswer)
  results: UserQuizResult | null; // Финальный результат пользователя
  leaderboard: UserQuizResult[] | null; // Таблица лидеров (полученная по WS или API)
  remainingTime: number | null; // Оставшееся время на текущий вопрос в секундах
  quizStatus: 'idle' | 'waiting' | 'active' | 'question_active' | 'question_ended' | 'ended' | 'cancelled'; // Статус викторины с точки зрения пользователя
  isLoading: boolean; // Индикатор загрузки связанных данных
  error: string | null; // Ошибка, связанная с состоянием викторины
  
  // Дополнительные поля для управления UI
  currentQuestionIndex: number | null; // Индекс текущего вопроса (для отслеживания прогресса)
  questionHistory: Question[]; // История вопросов для просмотра после завершения викторины
  isSubmitting: boolean; // Флаг отправки ответа
  hasSubmittedAnswer: boolean; // Флаг наличия ответа на текущий вопрос
} 