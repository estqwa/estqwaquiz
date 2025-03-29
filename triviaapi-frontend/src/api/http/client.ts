import axios, { AxiosInstance, AxiosRequestConfig, InternalAxiosRequestConfig } from 'axios';
import { store } from '../../store';
import { logoutSuccess } from '../../store/auth/slice';
import { transformKeysToSnakeCase, transformKeysToCamelCase } from '../../utils/api';

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
    withCredentials: true, // Важно для передачи HttpOnly cookies между доменами
    ...config,
  });

  // Интерцептор запроса для добавления CSRF токена и преобразования данных в snake_case
  client.interceptors.request.use(
    (config: InternalAxiosRequestConfig) => {
      const { csrfToken } = store.getState().auth;
      
      // Добавляем CSRF-токен для небезопасных методов, если он есть
      if (csrfToken && config.method && !['get', 'head', 'options'].includes(config.method.toLowerCase())) {
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
      // Если получили 401 Unauthorized, значит сессия истекла или пользователь не авторизован
      if (error.response && error.response.status === 401) {
        // Выход пользователя из системы
        store.dispatch(logoutSuccess());
        
        // Перенаправляем на страницу логина (если необходимо)
        // Можно добавить редирект на /login через router.push('/login') 
        // или через window.location.href = '/login'
      }

      // Для всех остальных ошибок просто пробрасываем их дальше
      return Promise.reject(error);
    }
  );

  return client;
};

// Экспортируем экземпляр клиента для использования в приложении
export const apiClient = createApiClient(); 