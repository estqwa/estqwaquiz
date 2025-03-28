/**
 * Интерфейс пользователя, соответствующий модели 'users' из базы данных
 */
export interface User {
  id: number;
  username: string;
  email: string;
  role: 'user' | 'admin' | 'moderator'; // Роль пользователя
  created_at: string;
  updated_at: string;
  avatar_url?: string; // URL аватара пользователя
  profile_picture?: string; // Альтернативное название аватара, иногда используемое в API
  is_active: boolean; // Статус активности учетной записи
  settings?: Record<string, any>; // Пользовательские настройки в формате JSON
  
  // Статистика пользователя
  games_played?: number;
  total_score?: number;
  highest_score?: number;
}