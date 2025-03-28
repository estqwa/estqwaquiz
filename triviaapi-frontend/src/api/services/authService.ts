import { apiClient } from '../http/client';
import { User } from '../../types/user';
import { store } from '../../store';

// Интерфейсы для запросов и ответов
export interface LoginRequest {
  email: string;
  password: string;
  token_based?: boolean; // Для Bearer Auth
}

export interface RegisterRequest {
  username: string;
  email: string;
  password: string;
  password_confirmation: string;
}

export interface AuthResponse {
  user: User;
  access_token?: string;    // Для Bearer Auth
  refresh_token?: string;   // Для Bearer Auth
  csrf_token?: string;      // Для Cookie Auth
  expires_in?: number;
}

export interface RefreshTokenResponse {
  accessToken: string | null;
  csrfToken?: string | null;
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
    // Определяем режим аутентификации
    const useCookieAuth = !credentials.token_based;
    
    // Удаляем token_based из запроса, так как бэкенд мог его не ожидать
    const { token_based, ...loginData } = credentials;
    
    const response = await apiClient.post<any>('/auth/login', loginData);
    
    // Проверяем различные форматы ответа от сервера
    if (response.data.success && response.data.data) {
      // Формат {success: true, data: {...}}
      return response.data.data;
    } else if (response.data.user) {
      // Формат ответа содержит user напрямую в data
      return response.data;
    } else {
      console.error('Unexpected response format:', response.data);
      throw new Error('Unexpected response format from the server');
    }
  },

  /**
   * Регистрация нового пользователя
   */
  register: async (userData: RegisterRequest): Promise<AuthResponse> => {
    const response = await apiClient.post<{success: boolean, data: AuthResponse}>('/auth/register', userData);
    return response.data.data;
  },

  /**
   * Обновление токена доступа с использованием refresh токена.
   * В зависимости от режима аутентификации (cookie или bearer),
   * refresh токен передается либо в httpOnly cookie, либо в теле запроса.
   */
  refreshToken: async (): Promise<RefreshTokenResponse> => {
    const { useCookieAuth } = store.getState().auth;
    let requestData = {};

    // Для Bearer Auth нужно передать refresh_token в теле запроса
    if (!useCookieAuth) {
      // В реальном приложении refresh_token был бы в localStorage или другом безопасном хранилище
      const refreshToken = localStorage.getItem('refresh_token');
      if (!refreshToken) {
        throw new Error('No refresh token available');
      }
      requestData = { refresh_token: refreshToken };
    }

    // Для Cookie Auth refresh_token и так будет в cookies, ничего передавать не нужно

    const response = await apiClient.post<any>('/auth/refresh', requestData);
    
    // Обработка ответа в зависимости от формата и режима аутентификации
    let accessToken: string | null = null;
    let csrfToken: string | null = null;
    
    if (response.data) {
      // Для Bearer Auth ожидаем access_token в теле ответа
      if (!useCookieAuth) {
        if (response.data.success && response.data.data) {
          accessToken = response.data.data.access_token || null;
          // Сохраняем новый access_token для Bearer Auth если он не null
          if (accessToken) {
            localStorage.setItem('access_token', accessToken);
          }
        } else if (response.data.access_token) {
          accessToken = response.data.access_token;
          // Сохраняем новый access_token для Bearer Auth если он не null
          if (accessToken) {
            localStorage.setItem('access_token', accessToken);
          }
        }
      }
      
      // CSRF-токен может приходить по-разному, проверяем все возможные места
      if (response.data.success && response.data.data) {
        csrfToken = response.data.data.csrf_token || null;
      } else if (response.data.csrf_token) {
        csrfToken = response.data.csrf_token;
      }
    }

    return { accessToken, csrfToken };
  },

  /**
   * Выход пользователя из системы
   */
  logout: async (): Promise<void> => {
    // Учитываем режим аутентификации
    const { useCookieAuth } = store.getState().auth;

    try {
      await apiClient.post('/auth/logout');
      
      // Для Bearer Auth удаляем токены из localStorage
      if (!useCookieAuth) {
        localStorage.removeItem('access_token');
        localStorage.removeItem('refresh_token');
      }
      // Для Cookie Auth cookie будут удалены бэкендом
    } catch (error) {
      console.error('Error during logout:', error);
      // Даже при ошибке все равно очищаем локальное хранилище
      if (!useCookieAuth) {
        localStorage.removeItem('access_token');
        localStorage.removeItem('refresh_token');
      }
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