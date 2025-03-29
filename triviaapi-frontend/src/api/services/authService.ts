import { apiClient } from '../http/client';
import { User } from '../../types/user';
import { store } from '../../store';
import { loginSuccess, logoutSuccess } from '../../store/auth/slice';

// Интерфейсы для запросов и ответов
export interface LoginRequest {
  email: string;
  password: string;
}

export interface RegisterRequest {
  username: string;
  email: string;
  password: string;
  password_confirmation: string;
}

export interface AuthResponse {
  user: User;
  csrf_token?: string;      // Для Cookie Auth
  expires_in?: number;
}

export interface WsTicketResponse {
  ticket: string;
}

// Сервис для работы с API аутентификации
export const authService = {
  /**
   * Вход пользователя в систему
   */
  login: async (credentials: LoginRequest): Promise<AuthResponse> => {
    const response = await apiClient.post<any>('/auth/login', credentials);
    
    let userData: User;
    let csrfToken: string | undefined = undefined;
    
    // Проверяем различные форматы ответа от сервера
    if (response.data.success && response.data.data) {
      // Формат {success: true, data: {...}}
      userData = response.data.data.user;
      csrfToken = response.data.data.csrf_token || undefined;
    } else if (response.data.user) {
      // Формат ответа содержит user напрямую в data
      userData = response.data.user;
      csrfToken = response.data.csrf_token || undefined;
    } else {
      console.error('Unexpected response format:', response.data);
      throw new Error('Unexpected response format from the server');
    }
    
    // Проверка наличия csrf_token в заголовках ответа
    if (!csrfToken && response.headers && response.headers['x-csrf-token']) {
      csrfToken = response.headers['x-csrf-token'];
    }
    
    // Диспатчим действие в store
    store.dispatch(loginSuccess({ user: userData, csrfToken: csrfToken || null }));
    
    return { user: userData, csrf_token: csrfToken };
  },

  /**
   * Регистрация нового пользователя
   */
  register: async (userData: RegisterRequest): Promise<AuthResponse> => {
    const response = await apiClient.post<{success: boolean, data: AuthResponse}>('/auth/register', userData);
    
    const user = response.data.data.user;
    const csrfToken = response.data.data.csrf_token || undefined;
    
    // Диспатчим действие в store
    store.dispatch(loginSuccess({ user, csrfToken: csrfToken || null }));
    
    return response.data.data;
  },

  /**
   * Выход пользователя из системы
   */
  logout: async (): Promise<void> => {
    try {
      await apiClient.post('/auth/logout');
      // Очищаем состояние в Redux
      store.dispatch(logoutSuccess());
    } catch (error) {
      console.error('Error during logout:', error);
      // Даже при ошибке очищаем состояние
      store.dispatch(logoutSuccess());
      throw error;
    }
  },

  /**
   * Получение данных текущего пользователя
   */
  getCurrentUser: async (): Promise<User> => {
    const response = await apiClient.get<{success: boolean, data: User}>('/users/me');
    return response.data.data;
  },

  /**
   * Проверка статуса аутентификации
   * Полезно при инициализации приложения для проверки наличия активной сессии
   */
  checkAuth: async (): Promise<boolean> => {
    try {
      const response = await apiClient.get<{success: boolean, data: User}>('/users/me');
      return response.data.success;
    } catch (error) {
      return false;
    }
  },

  /**
   * Получение временного тикета для WebSocket подключения
   */
  getWsTicket: async (): Promise<WsTicketResponse> => {
    try {
      const response = await apiClient.post<{success: boolean, data: WsTicketResponse}>('/auth/ws-ticket');
      if (response.data.success && response.data.data) {
        return response.data.data;
      } else {
        throw new Error('Unexpected response format from the server');
      }
    } catch (error) {
      console.error('Error getting WebSocket ticket:', error);
      throw error;
    }
  }
}; 