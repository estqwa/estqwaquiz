/**
 * Утилиты для работы с API
 */

/**
 * Нормализует URL для API запросов
 * @param path путь к API эндпоинту
 * @returns полный URL
 */
export const getApiUrl = (path: string): string => {
  const baseUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api';
  return `${baseUrl}${path.startsWith('/') ? path : `/${path}`}`;
};

/**
 * Преобразует объект параметров в строку запроса URL
 * @param params объект с параметрами
 * @returns строка запроса (без начального ?)
 */
export const buildQueryString = (params: Record<string, any>): string => {
  return Object.entries(params)
    .filter(([, value]) => value !== undefined && value !== null)
    .map(([key, value]) => {
      if (Array.isArray(value)) {
        return value
          .map(val => `${encodeURIComponent(key)}[]=${encodeURIComponent(val)}`)
          .join('&');
      }
      if (typeof value === 'object') {
        return Object.entries(value)
          .map(([subKey, subValue]) => 
            `${encodeURIComponent(key)}[${encodeURIComponent(subKey)}]=${encodeURIComponent(String(subValue))}`)
          .join('&');
      }
      return `${encodeURIComponent(key)}=${encodeURIComponent(String(value))}`;
    })
    .join('&');
};

/**
 * Объединяет фильтры для API запросов
 * @param filters объект с фильтрами
 * @returns объект с параметрами запроса в формате filter[key]=value
 */
export const buildApiFilters = (filters: Record<string, any>): Record<string, any> => {
  return Object.entries(filters)
    .filter(([, value]) => value !== undefined && value !== null && value !== '')
    .reduce((acc, [key, value]) => {
      acc[`filter[${key}]`] = value;
      return acc;
    }, {} as Record<string, any>);
};

/**
 * Форматирует ответ API для отображения ошибок
 * @param error объект ошибки API
 * @returns форматированное сообщение об ошибке
 */
export const formatApiError = (error: any): string => {
  if (!error) return 'Неизвестная ошибка';
  
  if (error.response && error.response.data) {
    const { data } = error.response;
    if (data.error && data.error.message) {
      return data.error.message;
    }
    if (data.message) {
      return data.message;
    }
    if (data.errors && Array.isArray(data.errors)) {
      return data.errors.map((err: any) => err.message || err).join('. ');
    }
  }
  
  if (error.message) {
    if (error.message === 'Network Error') {
      return 'Ошибка сети. Проверьте подключение к интернету.';
    }
    return error.message;
  }
  
  return 'Произошла непредвиденная ошибка';
}; 