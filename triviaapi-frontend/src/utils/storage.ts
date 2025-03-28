/**
 * Утилиты для работы с локальным хранилищем
 */

/**
 * Сохраняет данные в локальное хранилище с возможностью выбора типа хранилища
 * @param key ключ для хранения
 * @param value значение для сохранения
 * @param useSessionStorage использовать sessionStorage вместо localStorage (опционально)
 */
export const saveToStorage = (key: string, value: any, useSessionStorage: boolean = false): void => {
  try {
    const storage = useSessionStorage ? sessionStorage : localStorage;
    const serializedValue = typeof value === 'string' ? value : JSON.stringify(value);
    storage.setItem(key, serializedValue);
  } catch (error) {
    console.error(`Ошибка при сохранении в ${useSessionStorage ? 'sessionStorage' : 'localStorage'}:`, error);
  }
};

/**
 * Получает данные из локального хранилища с возможностью выбора типа хранилища
 * @param key ключ для получения данных
 * @param useSessionStorage использовать sessionStorage вместо localStorage (опционально)
 * @returns полученные данные или null в случае ошибки
 */
export const getFromStorage = <T>(key: string, useSessionStorage: boolean = false): T | null => {
  try {
    const storage = useSessionStorage ? sessionStorage : localStorage;
    const value = storage.getItem(key);
    if (value === null) return null;
    
    try {
      return JSON.parse(value) as T;
    } catch {
      return value as unknown as T;
    }
  } catch (error) {
    console.error(`Ошибка при получении из ${useSessionStorage ? 'sessionStorage' : 'localStorage'}:`, error);
    return null;
  }
};

/**
 * Удаляет данные из локального хранилища с возможностью выбора типа хранилища
 * @param key ключ для удаления
 * @param useSessionStorage использовать sessionStorage вместо localStorage (опционально)
 */
export const removeFromStorage = (key: string, useSessionStorage: boolean = false): void => {
  try {
    const storage = useSessionStorage ? sessionStorage : localStorage;
    storage.removeItem(key);
  } catch (error) {
    console.error(`Ошибка при удалении из ${useSessionStorage ? 'sessionStorage' : 'localStorage'}:`, error);
  }
};

/**
 * Очищает все данные из выбранного хранилища
 * @param useSessionStorage использовать sessionStorage вместо localStorage (опционально)
 */
export const clearStorage = (useSessionStorage: boolean = false): void => {
  try {
    const storage = useSessionStorage ? sessionStorage : localStorage;
    storage.clear();
  } catch (error) {
    console.error(`Ошибка при очистке ${useSessionStorage ? 'sessionStorage' : 'localStorage'}:`, error);
  }
};

/**
 * Хранит токены в зависимости от режима аутентификации
 * @param tokens объект с токенами
 * @param useCookieAuth режим cookie-based аутентификации
 */
export const storeAuthTokens = (
  tokens: { 
    access_token?: string; 
    refresh_token?: string; 
    csrf_token?: string;
    expires_in?: number;
  }, 
  useCookieAuth: boolean
): void => {
  if (useCookieAuth) {
    // В режиме Cookie-based auth сохраняем только CSRF токен
    if (tokens.csrf_token) {
      saveToStorage('csrf_token', tokens.csrf_token);
    }
  } else {
    // В режиме Bearer auth сохраняем оба токена
    if (tokens.access_token) {
      saveToStorage('access_token', tokens.access_token);
    }
    if (tokens.refresh_token) {
      saveToStorage('refresh_token', tokens.refresh_token);
    }
    
    // Сохраняем время истечения токена, если оно предоставлено
    if (tokens.expires_in) {
      const expiresAt = Date.now() + tokens.expires_in * 1000;
      saveToStorage('token_expires_at', expiresAt);
    }
  }
  
  // Сохраняем режим аутентификации
  saveToStorage('use_cookie_auth', useCookieAuth);
};

/**
 * Получает сохраненные токены авторизации
 * @returns объект с токенами и режимом аутентификации
 */
export const getAuthTokens = (): { 
  accessToken: string | null; 
  refreshToken: string | null; 
  csrfToken: string | null;
  useCookieAuth: boolean;
  expiresAt: number | null;
} => {
  const useCookieAuth = getFromStorage<boolean>('use_cookie_auth') ?? false;
  
  return {
    accessToken: getFromStorage<string>('access_token'),
    refreshToken: getFromStorage<string>('refresh_token'),
    csrfToken: getFromStorage<string>('csrf_token'),
    useCookieAuth,
    expiresAt: getFromStorage<number>('token_expires_at')
  };
};

/**
 * Удаляет все токены аутентификации
 */
export const clearAuthTokens = (): void => {
  removeFromStorage('access_token');
  removeFromStorage('refresh_token');
  removeFromStorage('csrf_token');
  removeFromStorage('token_expires_at');
  // Режим аутентификации можно сохранить для следующего входа
}; 