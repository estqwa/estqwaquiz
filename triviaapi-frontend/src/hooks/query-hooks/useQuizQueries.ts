import { useQuery, useMutation, useQueryClient, UseQueryOptions } from '@tanstack/react-query';
import { quizService } from '../../api/services/quizService';
import { Quiz } from '../../types/quiz';
import { Question } from '../../types/question';
import { UserQuizResult } from '../../types/result';
import { useCallback } from 'react';

// Ключи запросов для React Query
export const quizKeys = {
  all: ['quizzes'] as const,
  lists: (filters?: any) => [...quizKeys.all, 'list', { ...filters }] as const,
  details: (id: number) => [...quizKeys.all, 'detail', id] as const,
  active: () => [...quizKeys.all, 'active'] as const,
  scheduled: () => [...quizKeys.all, 'scheduled'] as const,
  results: (id: number) => [...quizKeys.details(id), 'results'] as const,
  userResult: (quizId: number, userId?: number) => 
    [...quizKeys.results(quizId), 'user', userId ?? 'current'] as const,
  questions: (id: number) => [...quizKeys.details(id), 'questions'] as const,
};

/**
 * Хук для получения списка викторин с пагинацией и фильтрацией
 */
export const useQuizzes = (page = 1, pageSize = 10, filters?: Record<string, any>) => {
  return useQuery({
    queryKey: quizKeys.lists({ page, pageSize, ...filters }),
    queryFn: () => quizService.getQuizzes(page, pageSize, filters),
    placeholderData: (previousData) => previousData,
    staleTime: 60 * 1000, // 1 минута
  });
};

/**
 * Хук для получения информации о конкретной викторине
 */
export const useQuiz = (id: number | undefined, options?: UseQueryOptions<Quiz | null, Error>) => {
  return useQuery({
    queryKey: quizKeys.details(id!),
    queryFn: () => quizService.getQuiz(id!),
    enabled: !!id,
    staleTime: 5 * 60 * 1000, // 5 минут
    ...options,
  });
};

/**
 * Хук для получения викторины вместе с вопросами
 */
export const useQuizWithQuestions = (
  id: number | undefined, 
  options?: UseQueryOptions<(Quiz & { questions: Question[] }) | null, Error>
) => {
  return useQuery({
    queryKey: quizKeys.questions(id!),
    queryFn: () => quizService.getQuizWithQuestions(id!),
    enabled: !!id,
    staleTime: 10 * 60 * 1000, // 10 минут
    ...options,
  });
};

/**
 * Хук для получения активных викторин
 */
export const useActiveQuizzes = (options?: UseQueryOptions<Quiz[], Error>) => {
  return useQuery({
    queryKey: quizKeys.active(),
    queryFn: quizService.getActiveQuizzes,
    staleTime: 15 * 1000, // 15 секунд
    refetchInterval: 30 * 1000, // Перепроверяем каждые 30 секунд
    ...options,
  });
};

/**
 * Хук для получения единственной активной викторины
 */
export const useActiveQuiz = (options?: UseQueryOptions<Quiz | null, Error>) => {
  return useQuery({
    queryKey: [...quizKeys.active(), 'single'],
    queryFn: async () => {
      const quizzes = await quizService.getActiveQuizzes();
      return quizzes.length > 0 ? quizzes[0] : null;
    },
    staleTime: 15 * 1000, // 15 секунд
    refetchInterval: 30 * 1000, // Перепроверяем каждые 30 секунд
    ...options,
  });
};

/**
 * Хук для получения запланированных викторин
 */
export const useScheduledQuizzes = (options?: UseQueryOptions<Quiz[], Error>) => {
  return useQuery({
    queryKey: quizKeys.scheduled(),
    queryFn: quizService.getScheduledQuizzes,
    staleTime: 5 * 60 * 1000, // 5 минут
    ...options,
  });
};

/**
 * Хук для получения ближайшей запланированной викторины
 */
export const useNextScheduledQuiz = (options?: UseQueryOptions<Quiz | null, Error>) => {
  return useQuery({
    queryKey: [...quizKeys.scheduled(), 'next'],
    queryFn: async () => {
      console.log('Fetching scheduled quizzes');
      try {
        const quizzes = await quizService.getScheduledQuizzes();
        console.log('Scheduled quizzes received:', quizzes);
        
        if (!quizzes || quizzes.length === 0) {
          console.log('No scheduled quizzes available');
          return null;
        }
        
        // Сортируем по времени начала (от ближайшего)
        const sortedQuizzes = quizzes.sort((a, b) => {
          const timeA = new Date(a.start_time || a.scheduled_time || '').getTime();
          const timeB = new Date(b.start_time || b.scheduled_time || '').getTime();
          return timeA - timeB;
        });
        
        console.log('Next scheduled quiz:', sortedQuizzes[0]);
        return sortedQuizzes[0];
      } catch (error) {
        console.error('Error fetching scheduled quizzes:', error);
        throw error;
      }
    },
    staleTime: 60 * 1000, // 1 минута
    refetchInterval: 60 * 1000, // Перепроверяем каждую минуту
    retry: 2,
    ...options,
  });
};

/**
 * Хук для получения результатов викторины
 */
export const useQuizResults = (
  quizId: number | undefined, 
  page = 1, 
  pageSize = 10,
  options?: UseQueryOptions<{ 
    results: UserQuizResult[];
    meta: { pagination: any };
  } | null, Error>
) => {
  return useQuery({
    queryKey: [...quizKeys.results(quizId!), { page, pageSize }],
    queryFn: () => quizService.getQuizResults(quizId!, page, pageSize),
    enabled: !!quizId,
    placeholderData: (previousData) => previousData,
    staleTime: 5 * 60 * 1000, // 5 минут
    ...options,
  });
};

/**
 * Хук для получения результата пользователя в викторине
 */
export const useUserQuizResult = (
  quizId: number | undefined, 
  userId?: number,
  options?: UseQueryOptions<UserQuizResult | null, Error>
) => {
  return useQuery({
    queryKey: quizKeys.userResult(quizId!, userId),
    queryFn: () => quizService.getUserQuizResult(quizId!, userId),
    enabled: !!quizId,
    staleTime: 5 * 60 * 1000, // 5 минут
    ...options,
  });
};

/**
 * Хук для получения таблицы лидеров викторины
 */
export const useQuizLeaderboard = (
  quizId: number | undefined,
  page = 1,
  pageSize = 10,
  options?: UseQueryOptions<{
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
    meta: { pagination: any };
  } | null, Error>
) => {
  return useQuery({
    queryKey: [...quizKeys.results(quizId!), 'leaderboard', { page, pageSize }],
    queryFn: () => quizService.getLeaderboard(quizId!, page, pageSize),
    enabled: !!quizId,
    staleTime: 30 * 1000, // 30 секунд
    ...options,
  });
};

/**
 * Хук для создания викторины
 */
export const useCreateQuiz = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (quizData: Partial<Quiz>) => quizService.createQuiz(quizData),
    onSuccess: () => {
      // Инвалидируем запросы списков после создания новой викторины
      queryClient.invalidateQueries({ queryKey: quizKeys.all });
    },
  });
};

/**
 * Хук для обновления викторины
 */
export const useUpdateQuiz = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: Partial<Quiz> }) => 
      quizService.updateQuiz(id, data),
    onSuccess: (updatedQuiz) => {
      // Обновляем кэш для конкретной викторины и списки
      if (updatedQuiz) {
        queryClient.setQueryData(quizKeys.details(updatedQuiz.id), updatedQuiz);
        queryClient.invalidateQueries({ queryKey: quizKeys.lists() });
      }
    },
  });
};

/**
 * Хук для удаления викторины
 */
export const useDeleteQuiz = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (id: number) => quizService.deleteQuiz(id),
    onSuccess: (_, id) => {
      // Удаляем викторину из кэша и инвалидируем списки
      queryClient.removeQueries({ queryKey: quizKeys.details(id) });
      queryClient.invalidateQueries({ queryKey: quizKeys.lists() });
    },
  });
};

/**
 * Хук для добавления вопросов к викторине
 */
export const useAddQuestions = (quizId: number | undefined) => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (questions: Partial<Question>[]) => 
      quizService.addQuestions(quizId!, questions),
    onSuccess: () => {
      // Инвалидируем запросы вопросов для данной викторины
      if (quizId) {
        queryClient.invalidateQueries({ queryKey: quizKeys.questions(quizId) });
        queryClient.invalidateQueries({ queryKey: quizKeys.details(quizId) });
      }
    }
  });
};

/**
 * Хук для планирования викторины
 */
export const useScheduleQuiz = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ quizId, startTime }: { quizId: number; startTime: string }) => 
      quizService.scheduleQuiz(quizId, startTime),
    onSuccess: (updatedQuiz) => {
      // Обновляем кэш и инвалидируем связанные запросы
      if (updatedQuiz) {
        queryClient.setQueryData(quizKeys.details(updatedQuiz.id), updatedQuiz);
        queryClient.invalidateQueries({ queryKey: quizKeys.scheduled() });
        queryClient.invalidateQueries({ queryKey: quizKeys.lists() });
      }
    },
  });
};

/**
 * Хук для отмены викторины
 */
export const useCancelQuiz = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (quizId: number) => quizService.cancelQuiz(quizId),
    onSuccess: (updatedQuiz) => {
      // Обновляем кэш и инвалидируем связанные запросы
      if (updatedQuiz) {
        queryClient.setQueryData(quizKeys.details(updatedQuiz.id), updatedQuiz);
        queryClient.invalidateQueries({ queryKey: quizKeys.scheduled() });
        queryClient.invalidateQueries({ queryKey: quizKeys.active() });
        queryClient.invalidateQueries({ queryKey: quizKeys.lists() });
      }
    },
  });
};

/**
 * Хук для присоединения к викторине
 */
export const useJoinQuiz = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ quizId, joinCode }: { quizId: number; joinCode?: string }) => 
      quizService.joinQuiz(quizId, joinCode),
    onSuccess: (_, { quizId }) => {
      // Инвалидируем данные о викторине после присоединения
      queryClient.invalidateQueries({ queryKey: quizKeys.details(quizId) });
    },
  });
};

/**
 * Хук для выхода из викторины
 */
export const useLeaveQuiz = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (quizId: number) => quizService.leaveQuiz(quizId),
    onSuccess: (_, quizId) => {
      // Инвалидируем данные о викторине после выхода
      queryClient.invalidateQueries({ queryKey: quizKeys.details(quizId) });
    },
  });
};

/**
 * Хук для предзагрузки деталей викторины
 * Полезно при навигации на страницу деталей
 */
export const usePrefetchQuiz = () => {
  const queryClient = useQueryClient();
  
  return useCallback((id: number) => {
    queryClient.prefetchQuery({
      queryKey: quizKeys.details(id),
      queryFn: () => quizService.getQuiz(id),
      staleTime: 5 * 60 * 1000, // 5 минут
    });
  }, [queryClient]);
}; 