/**
 * Утилиты для работы с WebSocket
 */
import { WebSocketMessage, WebSocketEventType } from '../types/websocket';

/**
 * Создает WebSocket-сообщение
 * @param type тип сообщения
 * @param data данные сообщения
 * @param priority приоритет сообщения (опционально)
 * @returns объект WebSocketMessage
 */
export const createWebSocketMessage = <T>(
  type: WebSocketEventType | string,
  data: T,
  priority: 'LOW' | 'NORMAL' | 'HIGH' | 'CRITICAL' = 'NORMAL'
): WebSocketMessage => {
  return {
    type,
    data,
    priority
  };
};

/**
 * Проверяет, является ли сообщение указанного типа
 * @param message WebSocket сообщение
 * @param type ожидаемый тип сообщения
 * @returns true, если тип сообщения совпадает с ожидаемым
 */
export const isMessageOfType = (message: WebSocketMessage, type: WebSocketEventType | string): boolean => {
  return message.type === type;
};

/**
 * Извлекает данные из WebSocket сообщения с типизацией
 * @param message WebSocket сообщение
 * @returns типизированные данные сообщения или null
 */
export const extractMessageData = <T>(message: WebSocketMessage): T | null => {
  return message.data as T;
};

/**
 * Вычисляет экспоненциальную задержку для повторного подключения
 * @param attempt номер попытки подключения
 * @param baseDelay базовая задержка (мс)
 * @param maxDelay максимальная задержка (мс)
 * @returns время задержки в миллисекундах
 */
export const calculateReconnectDelay = (
  attempt: number,
  baseDelay: number = 1000,
  maxDelay: number = 30000
): number => {
  // Экспоненциальная задержка: базовая_задержка * 2^(попытка - 1)
  const delay = baseDelay * Math.pow(2, attempt - 1);
  // Ограничиваем максимальную задержку
  return Math.min(delay, maxDelay);
};

/**
 * Проверяет, является ли сообщение критическим или высокоприоритетным
 * @param message WebSocket сообщение
 * @returns true, если сообщение имеет высокий приоритет
 */
export const isHighPriorityMessage = (message: WebSocketMessage): boolean => {
  return message.priority === 'CRITICAL' || message.priority === 'HIGH';
}; 