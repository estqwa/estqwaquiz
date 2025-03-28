/**
 * Утилиты для валидации данных
 */

/**
 * Проверяет корректность email адреса
 * @param email строка для проверки
 * @returns true, если email корректный
 */
export const isValidEmail = (email: string): boolean => {
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  return emailRegex.test(email);
};

/**
 * Проверяет минимальную длину строки
 * @param value строка для проверки
 * @param minLength минимальная длина
 * @returns true, если длина строки не меньше minLength
 */
export const hasMinLength = (value: string, minLength: number): boolean => {
  return value.length >= minLength;
};

/**
 * Проверяет максимальную длину строки
 * @param value строка для проверки
 * @param maxLength максимальная длина
 * @returns true, если длина строки не больше maxLength
 */
export const hasMaxLength = (value: string, maxLength: number): boolean => {
  return value.length <= maxLength;
};

/**
 * Проверяет, содержит ли пароль требуемые символы
 * @param password строка пароля для проверки
 * @returns true, если пароль соответствует требованиям
 */
export const isStrongPassword = (password: string): boolean => {
  // Минимум 8 символов, хотя бы одна заглавная буква, одна строчная буква, одна цифра и один спец. символ
  const passwordRegex = /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)(?=.*[^\da-zA-Z]).{8,}$/;
  return passwordRegex.test(password);
};

/**
 * Проверяет, соответствует ли username требованиям
 * @param username строка username для проверки
 * @returns true, если username соответствует требованиям
 */
export const isValidUsername = (username: string): boolean => {
  // Разрешены только буквы, цифры, подчеркивания, точки, дефисы
  // Длина от 3 до 20 символов
  const usernameRegex = /^[a-zA-Z0-9._-]{3,20}$/;
  return usernameRegex.test(username);
};

/**
 * Проверяет, является ли значение положительным числом
 * @param value значение для проверки
 * @returns true, если значение является положительным числом
 */
export const isPositiveNumber = (value: number): boolean => {
  return !isNaN(value) && value > 0;
};

/**
 * Проверяет, является ли значение целым положительным числом
 * @param value значение для проверки
 * @returns true, если значение является целым положительным числом
 */
export const isPositiveInteger = (value: number): boolean => {
  return !isNaN(value) && Number.isInteger(value) && value > 0;
};

/**
 * Проверяет, находится ли число в заданном диапазоне
 * @param value число для проверки
 * @param min минимальное значение
 * @param max максимальное значение
 * @returns true, если число находится в диапазоне [min, max]
 */
export const isInRange = (value: number, min: number, max: number): boolean => {
  return !isNaN(value) && value >= min && value <= max;
};

/**
 * Проверяет, не является ли строка пустой
 * @param value строка для проверки
 * @returns true, если строка не пустая
 */
export const isNotEmpty = (value: string): boolean => {
  return value.trim().length > 0;
};

/**
 * Проверяет, является ли строка URL
 * @param value строка для проверки
 * @returns true, если строка является корректным URL
 */
export const isValidUrl = (value: string): boolean => {
  try {
    new URL(value);
    return true;
  } catch {
    return false;
  }
}; 