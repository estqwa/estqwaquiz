/**
 * Утилиты для форматирования даты и времени
 */

/**
 * Форматирует дату в локальный формат
 * @param dateString ISO строка даты
 * @returns форматированная дата
 */
export const formatDate = (dateString: string): string => {
  const date = new Date(dateString);
  return date.toLocaleDateString();
};

/**
 * Форматирует дату и время в локальный формат
 * @param dateString ISO строка даты
 * @returns форматированная дата и время
 */
export const formatDateTime = (dateString: string): string => {
  const date = new Date(dateString);
  return date.toLocaleString();
};

/**
 * Форматирует время в локальный формат
 * @param dateString ISO строка даты
 * @returns форматированное время
 */
export const formatTime = (dateString: string): string => {
  const date = new Date(dateString);
  return date.toLocaleTimeString();
};

/**
 * Форматирует время в миллисекундах в формат "минуты:секунды"
 * @param ms время в миллисекундах
 * @returns строка в формате "минуты:секунды" (например, "2:45")
 */
export const formatMillisecondsToMinSec = (ms: number): string => {
  const totalSeconds = Math.floor(ms / 1000);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${minutes}:${seconds.toString().padStart(2, '0')}`;
};

/**
 * Преобразует секунды в строку с форматом времени
 * @param seconds количество секунд
 * @returns строка в формате "мин:сек" (например, "2:45")
 */
export const formatSecondsToMinSec = (seconds: number): string => {
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = Math.floor(seconds % 60);
  return `${minutes}:${remainingSeconds.toString().padStart(2, '0')}`;
};

/**
 * Вычисляет разницу между двумя датами в миллисекундах
 * @param startDate начальная дата (ISO строка)
 * @param endDate конечная дата (ISO строка)
 * @returns разница в миллисекундах
 */
export const getDateDifferenceInMs = (startDate: string, endDate: string): number => {
  const start = new Date(startDate).getTime();
  const end = new Date(endDate).getTime();
  return end - start;
};

/**
 * Проверяет, истекла ли дата
 * @param dateString ISO строка даты для проверки
 * @returns true, если дата уже прошла, иначе false
 */
export const isExpired = (dateString: string): boolean => {
  const date = new Date(dateString).getTime();
  const now = Date.now();
  return date < now;
}; 