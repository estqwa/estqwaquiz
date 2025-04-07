import { httpClient, ApiError } from './http-client';

// Интерфейс пользователя, получаемого с сервера
export interface User {
  id: number;
  username: string;
  email: string;
  profile_picture: string | null;
  games_played: number;
  total_score: number;
  highest_score: number;
}

// Интерфейс для ответа аутентификации
export interface AuthResponse {
  user: User;
  token_type: string;
  expires_in: number;
  csrf_token?: string;
}

// Интерфейс для запроса регистрации
export interface RegisterRequest {
  username: string;
  email: string;
  password: string;
}

// Интерфейс для запроса логина
export interface LoginRequest {
  email: string;
  password: string;
  device_id?: string;
}

// Интерфейс для информации о токене
export interface TokenInfo {
  user_id: number;
  username: string;
  email: string;
  is_admin: boolean;
  exp: number; // Время истечения токена
  csrf_token: string;
  access_token_expires: number;
  refresh_token_expires: number;
}

// Интерфейс для информации о сессии
export interface SessionInfo {
  id: number;
  device_id: string;
  ip_address: string;
  user_agent: string;
  created_at: string; // Даты будут строками ISO
  expires_at: string;
}

// Интерфейс для ответа со списком сессий
export interface SessionsResponse {
  sessions: SessionInfo[];
  count: number;
}

// Интерфейс для ответа с WebSocket тикетом
export interface WsTicketResponse {
  success: boolean;
  data: {
    ticket: string;
  };
}

/**
 * Регистрирует нового пользователя
 * 
 * @param data Данные для регистрации (username, email, password)
 * @returns Promise с данными пользователя и токенами
 * @throws ApiError в случае ошибки
 */
export async function registerUser(data: RegisterRequest): Promise<AuthResponse> {
  return httpClient.post<AuthResponse>('/api/auth/register', data);
}

/**
 * Аутентифицирует пользователя
 * 
 * @param email Email пользователя
 * @param password Пароль пользователя
 * @returns Promise с данными пользователя
 * @throws ApiError в случае ошибки
 */
export async function loginUser(email: string, password: string): Promise<AuthResponse> {
  return httpClient.post<AuthResponse>('/api/auth/login', { email, password });
}

/**
 * Обновляет токены аутентификации
 * Используя HttpOnly refreshToken куки
 * 
 * @param csrfToken CSRF-токен для защиты от CSRF-атак (из localStorage)
 * @returns Promise с новыми данными аутентификации (без токенов, т.к. они в куках)
 * @throws ApiError в случае ошибки
 */
export async function refreshTokens(csrfToken: string | null): Promise<Omit<AuthResponse, 'user'>> {
  const headers: Record<string, string> = {};
  
  if (csrfToken) {
    headers['X-CSRF-Token'] = csrfToken;
    console.log("refreshTokens: Отправка CSRF Token из аргумента:", csrfToken);
  } else {
    console.warn("refreshTokens: CSRF Token не передан!");
  }
  
  const response = await httpClient.post<Omit<AuthResponse, 'user'>>('/api/auth/refresh', undefined, { headers });
  return response;
}

/**
 * Выход из системы (логаут)
 * Отправляет запрос на сервер для инвалидации токенов
 * и очистки HttpOnly куки
 * 
 * @param csrfToken CSRF-токен для защиты от CSRF-атак (из localStorage)
 * @returns Promise<void>
 * @throws ApiError в случае ошибки
 */
export async function logoutUser(csrfToken: string | null): Promise<void> {
  const headers: Record<string, string> = {};
  
  if (csrfToken) {
    headers['X-CSRF-Token'] = csrfToken;
    console.log("logoutUser: Отправка CSRF Token из аргумента:", csrfToken);
  } else {
    console.error("Критическая ошибка: CSRF Token не передан в logoutUser!");
    throw new Error("CSRF Token не предоставлен для выхода.");
  }
  
  await httpClient.post<void>('/api/auth/logout', undefined, { headers });
}

/**
 * Получает информацию о токене
 * 
 * @returns Promise с информацией о токене
 */
export async function getTokenInfo(): Promise<TokenInfo> {
  return httpClient.get<TokenInfo>('/api/auth/token-info');
}

/**
 * Получает текущего аутентифицированного пользователя
 * 
 * @returns Promise с данными пользователя
 * @throws ApiError в случае ошибки
 */
export async function getCurrentUser(): Promise<User> {
  return httpClient.get<User>('/api/users/me');
}

/**
 * Проверяет, авторизован ли пользователь в настоящий момент
 * Использует куки и вызывает API для проверки
 * 
 * @returns Promise<boolean> - true если пользователь авторизован
 */
export async function isAuthenticated(): Promise<boolean> {
  try {
    await getCurrentUser();
    return true;
  } catch (error) {
    return false;
  }
}

/**
 * Получает специальный WebSocket тикет для подключения к WebSocket
 * Требует аутентифицированного пользователя с HttpOnly куками
 * 
 * @returns Promise с WebSocket тикетом
 * @throws ApiError в случае ошибки
 */
export async function getWebSocketTicket(): Promise<string> {
  try {
    const response = await httpClient.post<WsTicketResponse>('/api/auth/ws-ticket', {});
    return response.data.ticket;
  } catch (error) {
    console.error('Ошибка получения WebSocket тикета:', error);
    throw error;
  }
}

/**
 * Получает список активных сессий пользователя
 *
 * @returns Promise со списком сессий
 * @throws ApiError
 */
export async function getActiveSessions(): Promise<SessionsResponse> {
  return httpClient.get<SessionsResponse>('/api/auth/sessions');
} 