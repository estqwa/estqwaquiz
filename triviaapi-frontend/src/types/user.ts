/**
 * Интерфейс пользователя, соответствующий модели 'users' из базы данных
 */
export interface User {
  id: number;
  username: string;
  email: string;
  role: 'user' | 'admin' | 'moderator'; // Роль пользователя
  createdAt: string;
  updatedAt: string;
  avatarUrl?: string; // URL аватара пользователя
  profilePicture?: string; // Альтернативное название аватара, иногда используемое в API
  isActive: boolean; // Статус активности учетной записи
  settings?: Record<string, any>; // Пользовательские настройки в формате JSON
  
  // Статистика пользователя
  gamesPlayed?: number;
  totalScore?: number;
  highestScore?: number;
}