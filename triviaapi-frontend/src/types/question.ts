/**
 * Интерфейс вопроса, соответствующий модели из API
 */
export interface Question {
  id: number;
  quiz_id: number;
  text: string;
  type: 'single_choice' | 'multiple_choice' | 'text'; // Тип вопроса
  points: number; // Количество баллов за правильный ответ
  time_limit_seconds: number; // Ограничение времени ответа в секундах
  media_url?: string; // URL медиа-ресурса к вопросу
  options: Array<{ id: number; text: string }>; // Варианты ответов
  correct_answers?: number[] | string[]; // Правильные ответы (массив ID для multiple_choice или массив строк для text)
  hint?: string; // Подсказка к вопросу
  explanation?: string; // Объяснение правильного ответа
  order_num?: number; // Порядковый номер вопроса в викторине
  question_number?: number; // Альтернативное название для порядкового номера, используется в WS
  created_at?: string;
  updated_at?: string;
  
  // Дополнительные поля, приходящие от WebSocket
  duration_seconds?: number; // Синоним для time_limit_seconds, используется в WS
  total_questions?: number; // Общее количество вопросов в викторине
  start_time?: string; // Время начала вопроса
} 