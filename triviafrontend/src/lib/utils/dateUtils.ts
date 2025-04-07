// Форматы и стили для разных типов вывода дат
export enum DateFormat {
  SHORT = 'short',     // 01.01.2023, 14:30
  MEDIUM = 'medium',   // 1 января 2023, 14:30
  LONG = 'long',       // 1 января 2023 года, 14:30:00
  RELATIVE = 'relative' // 5 минут назад, через 2 часа
}

// Объединенные настройки форматирования для каждого типа
const formatOptions = {
  [DateFormat.SHORT]: {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  },
  [DateFormat.MEDIUM]: {
    day: 'numeric',
    month: 'long',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  },
  [DateFormat.LONG]: {
    day: 'numeric',
    month: 'long',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  }
};

/**
 * Парсит строку даты в объект Date
 * 
 * @param dateString строка даты (ISO 8601 или другой формат, распознаваемый Date)
 * @returns объект Date или null, если передана невалидная строка
 */
export function parseDate(dateString: string | undefined | null): Date | null {
  if (!dateString) return null;
  
  const date = new Date(dateString);
  // Проверяем, является ли дата валидной
  return isNaN(date.getTime()) ? null : date;
}

/**
 * Форматирует дату для отображения пользователю
 * 
 * @param dateString строка даты (ISO 8601 или другой формат, распознаваемый Date)
 * @param format формат вывода (short, medium, long)
 * @param locale локаль для форматирования (по умолчанию 'ru-RU')
 * @returns форматированная строка даты или 'Не указано' если передана невалидная строка
 */
export function formatDate(
  dateString: string | undefined | null, 
  format: DateFormat = DateFormat.MEDIUM,
  locale: string = 'ru-RU'
): string {
  const date = parseDate(dateString);
  if (!date) return 'Не указано';
  
  if (format === DateFormat.RELATIVE) {
    return formatRelativeDate(date);
  }
  
  return new Intl.DateTimeFormat(locale, formatOptions[format] as Intl.DateTimeFormatOptions).format(date);
}

/**
 * Форматирует относительную дату (например, "5 минут назад", "через 2 часа")
 * 
 * @param date объект Date
 * @returns строка относительной даты
 */
export function formatRelativeDate(date: Date): string {
  const now = new Date();
  const diffMs = date.getTime() - now.getTime();
  const diffSeconds = Math.round(diffMs / 1000);
  const diffMinutes = Math.round(diffSeconds / 60);
  const diffHours = Math.round(diffMinutes / 60);
  const diffDays = Math.round(diffHours / 24);
  
  // Если дата в прошлом
  if (diffMs < 0) {
    if (diffSeconds > -60) return 'только что';
    if (diffMinutes > -60) return `${Math.abs(diffMinutes)} ${getDeclension(Math.abs(diffMinutes), ['минуту', 'минуты', 'минут'])} назад`;
    if (diffHours > -24) return `${Math.abs(diffHours)} ${getDeclension(Math.abs(diffHours), ['час', 'часа', 'часов'])} назад`;
    if (diffDays > -30) return `${Math.abs(diffDays)} ${getDeclension(Math.abs(diffDays), ['день', 'дня', 'дней'])} назад`;
    
    // Если больше месяца назад, используем обычное форматирование
    return formatDate(date.toISOString(), DateFormat.MEDIUM);
  } 
  
  // Если дата в будущем
  if (diffSeconds < 60) return 'через несколько секунд';
  if (diffMinutes < 60) return `через ${diffMinutes} ${getDeclension(diffMinutes, ['минуту', 'минуты', 'минут'])}`;
  if (diffHours < 24) return `через ${diffHours} ${getDeclension(diffHours, ['час', 'часа', 'часов'])}`;
  if (diffDays < 30) return `через ${diffDays} ${getDeclension(diffDays, ['день', 'дня', 'дней'])}`;
  
  // Если больше месяца в будущем, используем обычное форматирование
  return formatDate(date.toISOString(), DateFormat.MEDIUM);
}

/**
 * Определяет правильное склонение существительного в зависимости от числа
 * 
 * @param number число
 * @param words массив форм слова [для 1, для 2-4, для 5-20]
 * @returns правильно склоненное слово
 */
export function getDeclension(number: number, words: [string, string, string]): string {
  const cases = [2, 0, 1, 1, 1, 2];
  return words[
    (number % 100 > 4 && number % 100 < 20) ? 2 : cases[Math.min(number % 10, 5)]
  ];
}

/**
 * Сравнивает две даты
 * 
 * @param date1 первая дата в виде строки или Date
 * @param date2 вторая дата в виде строки или Date
 * @returns отрицательное число если date1 < date2, 
 *          положительное если date1 > date2, 
 *          0 если равны
 */
export function compareDates(
  date1: string | Date | undefined | null, 
  date2: string | Date | undefined | null
): number {
  const d1 = typeof date1 === 'string' ? parseDate(date1) : date1;
  const d2 = typeof date2 === 'string' ? parseDate(date2) : date2;
  
  // Обработка null/undefined
  if (!d1 && !d2) return 0;
  if (!d1) return -1;
  if (!d2) return 1;
  
  return d1.getTime() - d2.getTime();
}

/**
 * Проверяет, находится ли дата в пределах заданного интервала от текущего момента
 * 
 * @param dateString строка даты
 * @param intervalMs интервал в миллисекундах
 * @returns true если дата находится в пределах интервала, иначе false
 */
export function isDateWithinInterval(dateString: string, intervalMs: number): boolean {
  const date = parseDate(dateString);
  if (!date) return false;
  
  const now = new Date();
  return Math.abs(date.getTime() - now.getTime()) <= intervalMs;
} 