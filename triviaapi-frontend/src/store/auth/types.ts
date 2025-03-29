// Типы для Auth Slice
import { User } from '../../types/user';

/**
 * Состояние авторизации для Redux-слайса
 */
export interface AuthState {
  user: User | null; // Данные пользователя
  csrfToken: string | null; // CSRF Token (используется при Cookie Auth)
  isAuthenticated: boolean; // Флаг аутентификации пользователя
  isLoading: boolean; // Флаг загрузки
  error: string | null; // Сообщение об ошибке
  status: 'idle' | 'loading' | 'succeeded' | 'failed' | 'checked'; // Статус проверки аутентификации
  expiresAt: number | null; // Время истечения токена в миллисекундах от эпохи
  activeSessions: Array<{
    device_info: string;
    ip_address: string;
    issued_at: string;
    is_current: boolean;
  }> | null; // Список активных сессий пользователя
} 