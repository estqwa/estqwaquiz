/**
 * Базовый HTTP клиент для работы с API
 * Настроен для отправки куки с каждым запросом
 */

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

// Типы HTTP методов
export type HttpMethod = 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH';

// Интерфейс для ошибок API
export interface ApiError {
  error: string;
  error_type?: string;
  status: number;
}

// Общие опции запроса
interface RequestOptions {
  headers?: Record<string, string>;
  query?: Record<string, string>;
}

/**
 * Читает значение cookie по имени.
 * @param name Имя cookie
 * @returns Значение cookie или null, если не найдено
 */
// УДАЛЕНО: function getCookie(...) - не используется для CSRF

/**
 * Выполняет HTTP запрос к API
 * 
 * @param method HTTP метод
 * @param endpoint Эндпоинт API без базового URL
 * @param body Тело запроса (для POST, PUT, PATCH)
 * @param options Дополнительные опции запроса
 * @returns Promise с данными ответа
 * @throws ApiError в случае ошибки
 */
export async function request<T>(
  method: HttpMethod,
  endpoint: string, 
  body?: any,
  options: RequestOptions = {}
): Promise<T> {
  // Формируем полный URL
  const url = new URL(`${API_BASE_URL}${endpoint}`);
  
  // Добавляем query параметры, если они есть
  if (options.query) {
    Object.entries(options.query).forEach(([key, value]) => {
      url.searchParams.append(key, value);
    });
  }

  // Базовые заголовки
  const headers: HeadersInit = {
    'Content-Type': 'application/json',
    ...options.headers,
  };

  // УДАЛЕНО: Логика добавления CSRF из cookie
  // Заголовок X-CSRF-Token должен добавляться вручную в options.headers при вызове request

  // Опции запроса
  const fetchOptions: RequestInit = {
    method,
    headers,
    credentials: 'include', // Важно для отправки и получения HttpOnly cookies
    body: body ? JSON.stringify(body) : undefined,
  };

  try {
    const response = await fetch(url.toString(), fetchOptions);
    
    // Если ответ не OK (статус не 2xx), выбрасываем ошибку
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({ 
        error: 'Ошибка при обработке ответа сервера' 
      }));
      
      throw {
        ...errorData,
        status: response.status,
      } as ApiError;
    }

    // Для ответов без содержимого (например, 204 No Content)
    if (response.status === 204) {
      return {} as T;
    }

    // Парсим JSON
    const data = await response.json();
    
    // ВОЗВРАЩЕНО: Логика поиска CSRF в теле ответа и сохранения в localStorage
    // Выводим полное содержимое ответа для отладки
    console.log("ПОЛУЧЕН ОТВЕТ ОТ СЕРВЕРА (data):", JSON.stringify(data, null, 2));
    
    // Проверяем наличие CSRF-токена в различных возможных полях
    const csrfFieldNames = ['csrf_token', 'csrfToken', 'csrf-token', 'xsrfToken', 'CSRF_TOKEN'];
    let foundCsrfToken = null;
    let tokenFieldName = null;
    
    if (data && typeof data === 'object') {
      // Ищем токен в корневых полях
      for (const fieldName of csrfFieldNames) {
        if (fieldName in data && data[fieldName]) {
          foundCsrfToken = data[fieldName];
          tokenFieldName = fieldName;
          break;
        }
      }
      
      // Если не нашли в корне, ищем вложенный объект (например, data.csrf_token)
      if (!foundCsrfToken && 'data' in data && typeof data.data === 'object') {
        for (const fieldName of csrfFieldNames) {
          if (fieldName in data.data && data.data[fieldName]) {
            foundCsrfToken = data.data[fieldName];
            tokenFieldName = `data.${fieldName}`;
            break;
          }
        }
      }
    }
    
    // Если нашли токен - сохраняем
    if (foundCsrfToken) {
      localStorage.setItem('csrf_token', foundCsrfToken);
      console.log(`CSRF Token НАЙДЕН в поле '${tokenFieldName}' и СОХРАНЕН в localStorage:`, foundCsrfToken);
    } else {
      console.log("CSRF Token НЕ НАЙДЕН ни в одном из проверяемых полей ответа сервера.");
    }
    
    return data as T;
  } catch (error) {
    // Перебрасываем ApiError или конвертируем обычное исключение в ApiError
    if ((error as ApiError).status) {
      throw error;
    }
    
    throw {
      error: (error as Error).message || 'Неизвестная ошибка',
      status: 0
    } as ApiError;
  }
}

// Вспомогательные методы для удобства
export const httpClient = {
  get: <T>(endpoint: string, options?: RequestOptions) => 
    request<T>('GET', endpoint, undefined, options),
    
  post: <T>(endpoint: string, body?: any, options?: RequestOptions) => 
    request<T>('POST', endpoint, body, options),
    
  put: <T>(endpoint: string, body?: any, options?: RequestOptions) => 
    request<T>('PUT', endpoint, body, options),
    
  delete: <T>(endpoint: string, body?: any, options?: RequestOptions) => 
    request<T>('DELETE', endpoint, body, options),
    
  patch: <T>(endpoint: string, body?: any, options?: RequestOptions) => 
    request<T>('PATCH', endpoint, body, options),
}; 