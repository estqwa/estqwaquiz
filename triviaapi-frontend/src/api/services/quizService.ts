import { apiClient } from '../http/client';
import { Quiz } from '../../types/quiz';
import { Question } from '../../types/question';
import { UserQuizResult } from '../../types/result';
import { buildApiFilters, buildQueryString } from '../../utils/api';

/**
 * Сервис для работы с API викторин
 */
export const quizService = {
  /**
   * Получение списка викторин с возможностью фильтрации и пагинации
   * @param page номер страницы
   * @param pageSize количество элементов на странице
   * @param filters фильтры для выборки
   * @returns список викторин и метаданные пагинации
   */
  getQuizzes: async (page = 1, pageSize = 10, filters?: Record<string, any>) => {
    const params = {
      page,
      per_page: pageSize,
      ...buildApiFilters(filters || {}),
    };

    try {
      const response = await apiClient.get(`/quizzes?${buildQueryString(params)}`);
      return response.data.data || [];
    } catch (error) {
      console.error('Failed to fetch quizzes:', error);
      return [];
    }
  },

  /**
   * Получение информации о конкретной викторине
   * @param id идентификатор викторины
   * @returns данные викторины
   */
  getQuiz: async (id: number): Promise<Quiz | null> => {
    try {
      const response = await apiClient.get(`/quizzes/${id}`);
      return response.data.data;
    } catch (error) {
      console.error(`Failed to fetch quiz #${id}:`, error);
      return null;
    }
  },

  /**
   * Получение информации о викторине вместе с вопросами
   * @param id идентификатор викторины
   * @returns данные викторины с вопросами
   */
  getQuizWithQuestions: async (id: number): Promise<(Quiz & { questions: Question[] }) | null> => {
    try {
      // Используем правильный эндпоинт в соответствии с API Reference
      const response = await apiClient.get(`/quizzes/${id}/with-questions`);
      return response.data.data;
    } catch (error) {
      console.error(`Failed to fetch quiz #${id} with questions:`, error);
      return null;
    }
  },

  /**
   * Получение списка активных викторин
   * @returns список активных викторин
   */
  getActiveQuizzes: async (): Promise<Quiz[]> => {
    try {
      const response = await apiClient.get('/quizzes/active');
      return response.data.data || [];
    } catch (error) {
      console.error('Failed to fetch active quizzes:', error);
      return [];
    }
  },

  /**
   * Получение списка запланированных викторин
   * @returns список запланированных викторин
   */
  getScheduledQuizzes: async (): Promise<Quiz[]> => {
    try {
      const response = await apiClient.get('/quizzes/scheduled');
      return response.data.data || [];
    } catch (error) {
      console.error('Failed to fetch scheduled quizzes:', error);
      return [];
    }
  },

  /**
   * Создание новой викторины
   * @param quizData данные для создания викторины
   * @returns созданная викторина
   */
  createQuiz: async (quizData: Partial<Quiz>): Promise<Quiz | null> => {
    try {
      const response = await apiClient.post('/quizzes', quizData);
      return response.data.data;
    } catch (error) {
      console.error('Failed to create quiz:', error);
      return null;
    }
  },

  /**
   * Обновление викторины
   * @param id идентификатор викторины
   * @param quizData данные для обновления
   * @returns обновленная викторина
   */
  updateQuiz: async (id: number, quizData: Partial<Quiz>): Promise<Quiz | null> => {
    try {
      const response = await apiClient.put(`/quizzes/${id}`, quizData);
      return response.data.data;
    } catch (error) {
      console.error(`Failed to update quiz #${id}:`, error);
      return null;
    }
  },

  /**
   * Удаление викторины
   * @param id идентификатор викторины
   * @returns результат операции
   */
  deleteQuiz: async (id: number): Promise<boolean> => {
    try {
      await apiClient.delete(`/quizzes/${id}`);
      return true;
    } catch (error) {
      console.error(`Failed to delete quiz #${id}:`, error);
      return false;
    }
  },

  /**
   * Добавление вопросов к викторине
   * @param quizId идентификатор викторины
   * @param questions массив вопросов для добавления
   * @returns обновленный список вопросов
   */
  addQuestions: async (quizId: number, questions: Partial<Question>[]): Promise<Question[] | null> => {
    try {
      const response = await apiClient.post(`/quizzes/${quizId}/questions`, { questions });
      return response.data.data;
    } catch (error) {
      console.error(`Failed to add questions to quiz #${quizId}:`, error);
      return null;
    }
  },

  /**
   * Запланировать викторину
   * @param quizId идентификатор викторины
   * @param startTime время начала (ISO строка)
   * @returns обновленная викторина
   */
  scheduleQuiz: async (quizId: number, startTime: string): Promise<Quiz | null> => {
    try {
      const response = await apiClient.put(`/quizzes/${quizId}/schedule`, { scheduled_time: startTime });
      return response.data.data;
    } catch (error) {
      console.error(`Failed to schedule quiz #${quizId}:`, error);
      return null;
    }
  },

  /**
   * Отменить викторину
   * @param quizId идентификатор викторины
   * @returns результат операции
   */
  cancelQuiz: async (quizId: number): Promise<Quiz | null> => {
    try {
      const response = await apiClient.put(`/quizzes/${quizId}/cancel`);
      return response.data.data;
    } catch (error) {
      console.error(`Failed to cancel quiz #${quizId}:`, error);
      return null;
    }
  },

  /**
   * Получение результатов викторины
   * @param quizId идентификатор викторины
   * @param page номер страницы
   * @param pageSize количество элементов на странице
   * @returns список результатов викторины
   */
  getQuizResults: async (quizId: number, page = 1, pageSize = 10): Promise<{
    results: UserQuizResult[];
    meta: {
      pagination: {
        total: number;
        per_page: number;
        current_page: number;
        last_page: number;
      }
    }
  } | null> => {
    try {
      const params = {
        page,
        per_page: pageSize
      };

      const response = await apiClient.get(`/quizzes/${quizId}/results?${buildQueryString(params)}`);
      return response.data;
    } catch (error) {
      console.error(`Failed to fetch results for quiz #${quizId}:`, error);
      return null;
    }
  },

  /**
   * Получение результата конкретного пользователя в викторине
   * @param quizId идентификатор викторины
   * @param userId идентификатор пользователя (опционально, по умолчанию текущий пользователь)
   * @returns результат пользователя
   */
  getUserQuizResult: async (quizId: number, userId?: number): Promise<UserQuizResult | null> => {
    try {
      const url = userId 
        ? `/quizzes/${quizId}/results/user/${userId}`
        : `/quizzes/${quizId}/my-result`;
      
      const response = await apiClient.get(url);
      return response.data.data;
    } catch (error) {
      console.error(`Failed to fetch user result for quiz #${quizId}:`, error);
      return null;
    }
  },

  /**
   * Получение результатов таблицы лидеров для викторины
   * @param quizId идентификатор викторины
   * @param page номер страницы
   * @param pageSize количество элементов на странице
   * @returns список результатов в таблице лидеров
   */
  getLeaderboard: async (quizId: number, page = 1, pageSize = 10): Promise<{
    data: Array<{
      rank: number;
      user: {
        id: number;
        username: string;
        avatar_url?: string;
      };
      score: number;
      correct_answers: number;
      total_questions: number;
      completion_time_ms: number;
    }>;
    meta: {
      pagination: {
        total: number;
        per_page: number;
        current_page: number;
        last_page: number;
      }
    }
  } | null> => {
    try {
      const params = {
        page,
        per_page: pageSize
      };

      const response = await apiClient.get(`/results/leaderboard/${quizId}?${buildQueryString(params)}`);
      return response.data;
    } catch (error) {
      console.error(`Failed to fetch leaderboard for quiz #${quizId}:`, error);
      return null;
    }
  },

  /**
   * Присоединение к викторине
   * @param quizId идентификатор викторины
   * @param joinCode код для присоединения (опционально, для приватных викторин)
   * @returns результат операции
   */
  joinQuiz: async (quizId: number, joinCode?: string): Promise<{
    quiz_id: number;
    participant_id: number;
    status: string;
    joined_at: string;
  } | null> => {
    try {
      const response = await apiClient.post(`/quizzes/${quizId}/join`, joinCode ? { join_code: joinCode } : {});
      return response.data.data;
    } catch (error) {
      console.error(`Failed to join quiz #${quizId}:`, error);
      return null;
    }
  },

  /**
   * Выход из викторины
   * @param quizId идентификатор викторины
   * @returns результат операции
   */
  leaveQuiz: async (quizId: number): Promise<boolean> => {
    try {
      await apiClient.post(`/quizzes/${quizId}/leave`);
      return true;
    } catch (error) {
      console.error(`Failed to leave quiz #${quizId}:`, error);
      return false;
    }
  },
}; 