/**
 * Интерфейс ответа пользователя, соответствующий модели 'user_answers' из базы данных и WebSocket API
 */
export interface UserAnswer {
  id?: number; // Уникальный идентификатор ответа
  user_id?: number; // Идентификатор пользователя
  quiz_id?: number; // Идентификатор викторины
  question_id: number; // Идентификатор вопроса
  option_id?: number; // ID выбранного ответа (используется в WebSocket)
  answer_data?: any; // Данные ответа пользователя в формате JSON
  is_correct?: boolean; // Флаг правильности ответа
  points_earned?: number; // Заработанные баллы
  answer_time_ms?: number; // Время ответа в миллисекундах
  time_taken_ms?: number; // Другое имя для времени ответа в WebSocket API
  submitted_at?: string; // Дата и время отправки ответа
  client_info?: Record<string, any>; // Информация о клиенте в формате JSON
  your_answer?: number; // Синоним для option_id в WebSocket API
} 