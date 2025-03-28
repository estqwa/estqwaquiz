/**
 * Интерфейс викторины, соответствующий модели из API
 */
export interface Quiz {
  id: number;
  title: string;
  description: string;
  category?: string; // Категория викторины
  difficulty?: 'easy' | 'medium' | 'hard'; // Сложность
  creator_id?: number; // ID создателя викторины
  is_public?: boolean; // Флаг публичной доступности
  start_time: string; // Запланированное время начала
  end_time?: string; // Запланированное время окончания
  duration_minutes?: number; // Продолжительность в минутах
  question_count: number; // Количество вопросов
  created_at?: string;
  updated_at?: string;
  settings?: Record<string, any>; // Настройки викторины в формате JSON
  status: 'draft' | 'published' | 'active' | 'completed' | 'cancelled'; // Статус викторины
  scheduled_time?: string; // Синоним для start_time, используется в некоторых API
} 