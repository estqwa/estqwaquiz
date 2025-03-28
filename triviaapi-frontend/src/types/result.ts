/**
 * Интерфейс результата викторины, соответствующий модели 'results' из базы данных
 */
export interface UserQuizResult {
  id: number; // Уникальный идентификатор результата
  userId: number; // Идентификатор пользователя
  quizId: number; // Идентификатор викторины
  username?: string; // Имя пользователя (для отображения в таблице лидеров)
  avatarUrl?: string; // URL аватара пользователя (для отображения)
  profilePicture?: string; // Альтернативное поле для аватара
  totalPoints: number; // Общее количество баллов
  correctAnswers: number; // Количество правильных ответов
  totalQuestions: number; // Общее количество вопросов в викторине
  completionTimeMs: number; // Время прохождения в миллисекундах
  rank: number; // Место в рейтинге
  completedAt: string; // Дата и время завершения
  detailedResults?: Record<string, any>; // Детальная статистика в формате JSON
  
  // Удобное поле для совместимости
  score?: number; // Алиас для totalPoints
} 