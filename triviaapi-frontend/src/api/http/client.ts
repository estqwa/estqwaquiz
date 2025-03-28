import axios, { AxiosInstance, AxiosRequestConfig, InternalAxiosRequestConfig } from 'axios';
import { store } from '../../store';
import { tokenRefreshed, logoutSuccess } from '../../store/auth/slice';
import { authService } from '../services/authService';
import { transformKeysToSnakeCase, transformKeysToCamelCase } from '../../utils/api';

let isRefreshing = false;
let failedQueue: Array<{ resolve: (value: unknown) => void; reject: (reason?: any) => void }> = [];

const processQueue = (error: any, token: string | null = null) => {
  failedQueue.forEach(prom => {
    if (error) {
      prom.reject(error);
    } else {
      prom.resolve(token);
    }
  });
  failedQueue = [];
};

// URL для API
const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api';

// Функция для создания HTTP клиента
export const createApiClient = (config?: AxiosRequestConfig): AxiosInstance => {
  const client = axios.create({
    baseURL: API_URL,
    timeout: 15000,
    headers: {
      'Content-Type': 'application/json',
      ...config?.headers,
    },
    withCredentials: true,
    ...config,
  });

  // Интерцептор запроса для добавления авторизационного токена и преобразования данных в snake_case
  client.interceptors.request.use(
    (config: InternalAxiosRequestConfig) => {
      const { token, csrfToken } = store.getState().auth;

      // Добавляем токен авторизации, если он есть
      if (token) {
        config.headers.Authorization = `Bearer ${token}`;
      }
      
      // Добавляем CSRF-токен, если он есть
      if (csrfToken) {
        config.headers['X-CSRF-Token'] = csrfToken;
      }

      // Преобразуем данные из camelCase в snake_case перед отправкой на сервер
      if (config.data) {
        config.data = transformKeysToSnakeCase(config.data);
      }

      return config;
    },
    (error) => Promise.reject(error)
  );

  // Интерцептор ответа для обработки ошибок и преобразования данных в camelCase
  client.interceptors.response.use(
    (response) => {
      // Преобразуем данные из snake_case в camelCase при получении от сервера
      if (response.data) {
        console.log('[Axios Interceptor] Response data BEFORE transform:', JSON.stringify(response.data));
        response.data = transformKeysToCamelCase(response.data);
        console.log('[Axios Interceptor] Response data AFTER transform:', JSON.stringify(response.data));
      }
      return response;
    },
    async (error) => {
      const originalRequest = error.config;
      
      // Проверяем, что ошибка - это 401 (Unauthorized) и запрос не является 
      // запросом на обновление токена (чтобы избежать бесконечного цикла)
      if (error.response && 
          error.response.status === 401 && 
          !originalRequest._retry && 
          !originalRequest.url?.includes('/auth/refresh')) {
        
        // Если уже идет процесс обновления токена, добавляем запрос в очередь
        if (isRefreshing) {
          return new Promise((resolve, reject) => {
            failedQueue.push({ resolve, reject });
          })
            .then(token => {
              // Обновляем заголовок Authorization для оригинального запроса
              if (token) {
                originalRequest.headers.Authorization = `Bearer ${token}`;
              }
              return client(originalRequest);
            })
            .catch(err => {
              return Promise.reject(err);
            });
        }

        // Устанавливаем флаг для предотвращения повторных попыток обновления
        originalRequest._retry = true;
        isRefreshing = true;

        try {
          // Пытаемся обновить токен
          const { accessToken, csrfToken } = await authService.refreshToken();
          
          // Сохраняем новые токены в Redux store
          store.dispatch(tokenRefreshed({ token: accessToken, csrfToken }));
          
          // Обрабатываем очередь неудавшихся запросов
          processQueue(null, accessToken);
          
          // Обновляем заголовки для оригинального запроса
          if (accessToken) {
            originalRequest.headers.Authorization = `Bearer ${accessToken}`;
          }
          
          if (csrfToken) {
            originalRequest.headers['X-CSRF-Token'] = csrfToken;
          }
          
          // Сбрасываем флаг обновления токена
          isRefreshing = false;
          
          // Повторяем оригинальный запрос с новыми токенами
          return client(originalRequest);
        } catch (refreshError) {
          // В случае ошибки обновления токена
          // Выход пользователя из системы
          store.dispatch(logoutSuccess());
          
          // Обрабатываем очередь с ошибкой
          processQueue(refreshError, null);
          
          // Сбрасываем флаг обновления токена
          isRefreshing = false;
          
          // Пробрасываем ошибку обновления токена дальше
          return Promise.reject(refreshError);
        }
      }

      // Для всех остальных ошибок просто пробрасываем их дальше
      return Promise.reject(error);
    }
  );

  return client;
};

// Экспортируем экземпляр клиента для использования в приложении
export const apiClient = createApiClient(); 