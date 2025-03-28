/**
 * Интерфейс викторины, соответствующий модели из API
 */
export interface Quiz {
  id: number;
  title: string;
  description: string;
  category?: string; // Категория викторины
  difficulty?: 'easy' | 'medium' | 'hard'; // Сложность
  creatorId?: number; // ID создателя викторины
  isPublic?: boolean; // Флаг публичной доступности
  startTime: string; // Запланированное время начала
  endTime?: string; // Запланированное время окончания
  durationMinutes?: number; // Продолжительность в минутах
  questionCount: number; // Количество вопросов
  createdAt?: string;
  updatedAt?: string;
  settings?: Record<string, any>; // Настройки викторины в формате JSON
  status: 'draft' | 'published' | 'active' | 'completed' | 'cancelled'; // Статус викторины
  scheduledTime?: string; // Синоним для startTime, используется в некоторых API
} 