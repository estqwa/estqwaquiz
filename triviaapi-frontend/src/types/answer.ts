/**
 * Интерфейс ответа пользователя, соответствующий модели 'user_answers' из базы данных и WebSocket API
 */
export interface UserAnswer {
  id?: number; // Уникальный идентификатор ответа
  userId?: number; // Идентификатор пользователя
  quizId?: number; // Идентификатор викторины
  questionId: number; // Идентификатор вопроса
  optionId?: number; // ID выбранного ответа (используется в WebSocket)
  answerData?: any; // Данные ответа пользователя в формате JSON
  isCorrect?: boolean; // Флаг правильности ответа
  pointsEarned?: number; // Заработанные баллы
  answerTimeMs?: number; // Время ответа в миллисекундах
  timeTakenMs?: number; // Другое имя для времени ответа в WebSocket API
  submittedAt?: string; // Дата и время отправки ответа
  clientInfo?: Record<string, any>; // Информация о клиенте в формате JSON
  yourAnswer?: number; // Синоним для optionId в WebSocket API
} 