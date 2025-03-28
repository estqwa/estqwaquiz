import React from 'react';
import { AxiosError } from 'axios';

interface ApiErrorAlertProps {
  error: unknown;
  className?: string;
  onRetry?: () => void;
}

/**
 * Компонент для отображения ошибок API в унифицированном формате
 */
const ApiErrorAlert: React.FC<ApiErrorAlertProps> = ({ error, className = '', onRetry }) => {
  // Обработка разных типов ошибок
  const getErrorMessage = (): { title: string; message: string } => {
    // Ошибки Axios
    if (error instanceof Error) {
      if (error instanceof AxiosError) {
        const axiosError = error as AxiosError<any>;
        
        // Проверяем наличие ответа
        if (axiosError.response) {
          const status = axiosError.response.status;
          
          // Обрабатываем различные коды состояния
          switch (status) {
            case 400:
              return {
                title: 'Некорректный запрос',
                message: axiosError.response.data?.message || 'Проверьте введенные данные и попробуйте снова.'
              };
            case 401:
              return {
                title: 'Требуется авторизация',
                message: 'Вы не авторизованы или срок действия сессии истек. Пожалуйста, войдите в систему.'
              };
            case 403:
              return {
                title: 'Доступ запрещен',
                message: 'У вас нет прав для выполнения этого действия.'
              };
            case 404:
              return {
                title: 'Ресурс не найден',
                message: 'Запрошенный ресурс не существует или был удален.'
              };
            case 422:
              return {
                title: 'Ошибка валидации',
                message: axiosError.response.data?.message || 'Проверьте правильность введенных данных.'
              };
            case 429:
              return {
                title: 'Слишком много запросов',
                message: 'Пожалуйста, попробуйте снова через несколько минут.'
              };
            case 500:
            case 502:
            case 503:
            case 504:
              return {
                title: 'Ошибка сервера',
                message: 'Произошла ошибка на сервере. Пожалуйста, попробуйте позже.'
              };
            default:
              return {
                title: `Ошибка (${status})`,
                message: axiosError.response.data?.message || axiosError.message || 'Произошла неизвестная ошибка.'
              };
          }
        }
        
        // Ошибка сети (например, сервер недоступен)
        if (axiosError.code === 'ECONNABORTED') {
          return {
            title: 'Тайм-аут запроса',
            message: 'Сервер не ответил вовремя. Пожалуйста, проверьте ваше подключение к интернету и попробуйте снова.'
          };
        }
        
        if (axiosError.code === 'ERR_NETWORK') {
          return {
            title: 'Ошибка сети',
            message: 'Не удалось подключиться к серверу. Пожалуйста, проверьте ваше подключение к интернету.'
          };
        }
        
        return {
          title: 'Ошибка запроса',
          message: axiosError.message || 'Произошла ошибка при отправке запроса.'
        };
      }
      
      // Обычные ошибки JavaScript
      return {
        title: 'Ошибка',
        message: error.message || 'Произошла неизвестная ошибка.'
      };
    }
    
    // Если ошибка - это строка
    if (typeof error === 'string') {
      return {
        title: 'Ошибка',
        message: error
      };
    }
    
    // Ошибка неизвестного типа
    return {
      title: 'Неизвестная ошибка',
      message: 'Произошла неизвестная ошибка. Пожалуйста, попробуйте позже.'
    };
  };
  
  // Если ошибки нет, ничего не показываем
  if (!error) {
    return null;
  }
  
  const { title, message } = getErrorMessage();
  
  return (
    <div className={`bg-red-50 p-4 rounded-md ${className}`}>
      <div className="flex">
        <div className="flex-shrink-0">
          <svg className="h-5 w-5 text-red-400" viewBox="0 0 20 20" fill="currentColor">
            <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
          </svg>
        </div>
        <div className="ml-3">
          <h3 className="text-sm font-medium text-red-800">{title}</h3>
          <div className="mt-2 text-sm text-red-700">
            <p>{message}</p>
          </div>
          
          {onRetry && (
            <div className="mt-4">
              <button
                type="button"
                onClick={onRetry}
                className="inline-flex items-center px-3 py-2 border border-transparent text-sm leading-4 font-medium rounded-md text-red-700 bg-red-100 hover:bg-red-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500"
              >
                Попробовать снова
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default ApiErrorAlert; 