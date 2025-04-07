import { httpClient } from './http-client';

// Интерфейс для объекта викторины
export interface Quiz {
  id: number;
  title: string;
  description: string;
  scheduled_time: string;
  status: 'scheduled' | 'in_progress' | 'completed';
  question_count: number;
  created_at: string;
  updated_at: string;
}

// Интерфейс для вопроса викторины
export interface Question {
  id: number;
  quiz_id: number;
  text: string;
  options: OptionData[];
  time_limit_sec: number;
  point_value: number;
  created_at: string;
  updated_at: string;
}

// Интерфейс для опции ответа (из WS)
export interface OptionData {
  id: number;
  text: string;
}

// Интерфейс для викторины с вопросами
export interface QuizWithQuestions extends Quiz {
  questions: Question[];
}

// Интерфейс для результата викторины одного пользователя
export interface QuizResult {
  id: number;
  user_id: number;
  quiz_id: number;
  username: string;
  profile_picture: string | null;
  score: number;
  rank: number;
  correct_answers: number;
  total_questions: number;
  is_eliminated: boolean;
  is_winner: boolean;
  prize_fund: number; // Сумма приза (может быть 0)
  completed_at: string;
}

// Интерфейс для параметров пагинации
export interface PaginationParams {
  page?: number;
  page_size?: number;
}

/**
 * Получает список доступных викторин
 * 
 * @param params параметры пагинации (опционально)
 * @returns Promise со списком викторин
 */
export async function getAvailableQuizzes(params?: PaginationParams): Promise<Quiz[]> {
  const queryParams: Record<string, string> = {};
  
  if (params?.page) {
    queryParams.page = params.page.toString();
  }
  
  if (params?.page_size) {
    queryParams.page_size = params.page_size.toString();
  }
  
  return httpClient.get<Quiz[]>('/api/quizzes', { query: queryParams });
}

/**
 * Получает информацию о конкретной викторине
 * 
 * @param quizId ID викторины
 * @returns Promise с информацией о викторине
 */
export async function getQuizById(quizId: number): Promise<Quiz> {
  return httpClient.get<Quiz>(`/api/quizzes/${quizId}`);
}

/**
 * Получает информацию о викторине с вопросами
 * 
 * @param quizId ID викторины
 * @returns Promise с информацией о викторине и ее вопросами
 */
export async function getQuizWithQuestions(quizId: number): Promise<QuizWithQuestions> {
  return httpClient.get<QuizWithQuestions>(`/api/quizzes/${quizId}/with-questions`);
}

/**
 * Получает активную викторину
 * 
 * @returns Promise с информацией об активной викторине
 */
export async function getActiveQuiz(): Promise<Quiz | null> {
  try {
    return await httpClient.get<Quiz>('/api/quizzes/active');
  } catch (error) {
    // Если нет активной викторины, вернем null
    return null;
  }
}

/**
 * Получает запланированные викторины
 * 
 * @returns Promise со списком запланированных викторин
 */
export async function getScheduledQuizzes(): Promise<Quiz[]> {
  return httpClient.get<Quiz[]>('/api/quizzes/scheduled');
}

/**
 * Получает результаты пользователя для конкретной викторины
 * 
 * @param quizId ID викторины
 * @returns Promise с результатами пользователя
 */
export async function getUserQuizResult(quizId: number): Promise<QuizResult | null> {
  try {
    return await httpClient.get<QuizResult>(`/api/quizzes/${quizId}/my-result`);
  } catch (error) {
    // Если результата нет (404), вернем null
    // TODO: Добавить более точную проверку кода ошибки
    return null;
  }
}

/**
 * Получает результаты викторины
 * 
 * @param quizId ID викторины
 * @returns Promise с результатами викторины
 */
export async function getQuizResults(quizId: number): Promise<QuizResult[]> {
  return httpClient.get<QuizResult[]>(`/api/quizzes/${quizId}/results`);
} 