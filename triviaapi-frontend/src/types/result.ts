/**
 * Интерфейс результата викторины, соответствующий модели 'results' из базы данных
 */
export interface UserQuizResult {
  id: number; // Уникальный идентификатор результата
  user_id: number; // Идентификатор пользователя
  quiz_id: number; // Идентификатор викторины
  username?: string; // Имя пользователя (для отображения в таблице лидеров)
  avatar_url?: string; // URL аватара пользователя (для отображения)
  profile_picture?: string; // Альтернативное поле для аватара
  total_points: number; // Общее количество баллов
  correct_answers: number; // Количество правильных ответов
  total_questions: number; // Общее количество вопросов в викторине
  completion_time_ms: number; // Время прохождения в миллисекундах
  rank: number; // Место в рейтинге
  completed_at: string; // Дата и время завершения
  detailed_results?: Record<string, any>; // Детальная статистика в формате JSON
  
  // Удобное поле для совместимости
  score?: number; // Алиас для total_points
} 