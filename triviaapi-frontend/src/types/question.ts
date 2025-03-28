/**
 * Интерфейс вопроса, соответствующий модели из API
 */
export interface Question {
  id: number;
  quizId: number;
  text: string;
  type: 'single_choice' | 'multiple_choice' | 'text'; // Тип вопроса
  points: number; // Количество баллов за правильный ответ
  timeLimitSeconds: number; // Ограничение времени ответа в секундах
  mediaUrl?: string; // URL медиа-ресурса к вопросу
  options: Array<{ id: number; text: string }>; // Варианты ответов
  correctAnswers?: number[] | string[]; // Правильные ответы (массив ID для multiple_choice или массив строк для text)
  hint?: string; // Подсказка к вопросу
  explanation?: string; // Объяснение правильного ответа
  orderNum?: number; // Порядковый номер вопроса в викторине
  questionNumber?: number; // Альтернативное название для порядкового номера, используется в WS
  createdAt?: string;
  updatedAt?: string;
  
  // Дополнительные поля, приходящие от WebSocket
  durationSeconds?: number; // Синоним для timeLimitSeconds, используется в WS
  totalQuestions?: number; // Общее количество вопросов в викторине
  startTime?: string; // Время начала вопроса
} 