import { httpClient } from './http-client';

// Интерфейс для результата пользователя в викторине
export interface Result {
  id: number;
  user_id: number;
  quiz_id: number;
  quiz_title?: string; // Название викторины (опционально)
  username: string;
  profile_picture: string;
  score: number;
  correct_answers: number;
  total_questions: number;
  rank: number;
  is_winner: boolean;
  prize_fund: number;
  completed_at: string;
  created_at: string;
  answers?: Array<{
    question_text: string;
    user_answer: string;
    correct_answer: string;
    is_correct: boolean;
  }>;
}

// Интерфейс для результата викторины (одного пользователя)
export interface QuizResult {
  id: number;
  user_id: number;
  quiz_id: number;
  username: string;
  profile_picture: string | null;
  score: number;
  correct_answers: number;
  total_questions: number;
  rank: number;
  is_winner: boolean;
  prize_fund: number;
  completed_at: string;
}

/**
 * Получает результаты конкретной викторины (для лидерборда)
 * 
 * @param quizId ID викторины
 * @returns Promise со списком результатов
 */
export async function getQuizResults(quizId: number): Promise<Result[]> {
  return httpClient.get<Result[]>(`/api/quizzes/${quizId}/results`);
}

/**
 * Получает результат текущего пользователя для конкретной викторины
 * 
 * @param quizId ID викторины
 * @returns Promise с результатом пользователя
 */
export async function getUserQuizResult(quizId: number): Promise<Result> {
  return httpClient.get<Result>(`/api/quizzes/${quizId}/my-result`);
}

// Удалены функции getQuizResult и getUserResults, т.к. их эндпоинты не существуют на бэкенде 