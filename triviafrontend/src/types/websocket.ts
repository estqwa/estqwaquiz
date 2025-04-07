// Типы WebSocket сообщений для викторины

// --- Сообщения от Клиента --- (Соответствуют бэкенду)

// Базовый интерфейс для всех исходящих сообщений
export interface WsClientMessageBase<T extends string, D> {
  type: T;
  data: D;
}

// Данные для готовности пользователя к викторине
export interface UserReadyData {
  quiz_id: number;
}

// Данные для отправки ответа
export interface UserAnswerData {
  question_id: number;
  selected_option: number; // Соответствует бэкенду
  timestamp: number;
}

// Данные для проверки соединения (пустые)
export interface UserHeartbeatData {
  // Пустой объект
}

// Типизированные сообщения от клиента
export type WsUserReadyMessage = WsClientMessageBase<'user:ready', UserReadyData>;
export type WsUserAnswerMessage = WsClientMessageBase<'user:answer', UserAnswerData>;
export type WsUserHeartbeatMessage = WsClientMessageBase<'user:heartbeat', UserHeartbeatData>;

// Объединенный тип для всех исходящих сообщений
export type WsClientMessage =
  | WsUserReadyMessage
  | WsUserAnswerMessage
  | WsUserHeartbeatMessage;

// --- Сообщения от Сервера --- (Приводим в соответствие с реальной отправкой бэкенда)

// Параметры опции вопроса (из API)
export interface OptionData {
  id: number; // Индекс + 1
  text: string;
}

// Данные для старта викторины (Тип: quiz:start)
export interface QuizStartData {
  quiz_id: number;
  title: string;
  question_count: number;
  // start_time: number; // Бэкенд НЕ отправляет это поле
}

// Данные для вопроса (Тип: quiz:question)
export interface QuizQuestionData {
  question_id: number;
  quiz_id: number;
  number: number;
  text: string;
  options: OptionData[]; // Соответствует (благодаря ConvertOptionsToObjects)
  time_limit: number;
  total_questions: number;
  start_time: number;
  server_timestamp: number;
}

// Данные для результата ответа (Тип: quiz:answer_result)
export interface QuizAnswerResultData {
  question_id: number;
  correct_option: number; // Бэкенд отправляет это
  your_answer: number;    // Бэкенд отправляет это
  is_correct: boolean;
  points_earned: number;
  time_taken_ms: number;
  is_eliminated: boolean; // Бэкенд отправляет это
  time_limit_exceeded: boolean; // Бэкенд отправляет это
}

// Данные для обновления таймера (Тип: quiz:timer)
export interface QuizTimerData {
  question_id: number;
  remaining_seconds: number;
  server_timestamp: number;
}

// Данные для показа правильного ответа (Тип: quiz:answer_reveal)
export interface QuizAnswerRevealData {
  question_id: number;
  correct_option: number;
}

// Данные для исключения пользователя (Тип: quiz:elimination)
export interface QuizEliminationData {
  message: string; // Бэкенд отправляет только это
  reason: string;  // и это
}

// Данные для напоминания об исключении (Тип: quiz:elimination_reminder)
export interface QuizEliminationReminderData {
  message: string;
  question_id: number;
}

// Данные о готовности пользователя (Тип: quiz:user_ready)
export interface QuizUserReadyData {
  user_id: number;
  quiz_id: number;
  status: string; // 'ready'
}

// Данные о завершении викторины (Тип: quiz:finish)
export interface QuizFinishData {
  quiz_id: number;
  title: string;
  message: string;
  status: string; // 'completed'
  ended_at: string; // ISO timestamp
}

// Данные о доступности результатов (Тип: quiz:results_available)
export interface QuizResultsAvailableData {
  quiz_id: number;
  message: string;
}

// Данные для ответа на heartbeat (Тип: server:heartbeat)
export interface ServerHeartbeatData {
  timestamp: number;
}

// Данные для ошибки (Тип: error)
export interface ErrorData {
  message: string;
  code?: string;
  critical?: boolean;
}

// Данные для анонса (Тип: quiz:announcement)
export interface QuizAnnouncementData {
  quiz_id: number;
  title: string;
  description: string;
  scheduled_time: string; // ISO timestamp
  question_count: number;
  minutes_to_start: number;
}

// Данные для зала ожидания (Тип: quiz:waiting_room)
export interface QuizWaitingRoomData {
  quiz_id: number;
  title: string;
  description: string;
  scheduled_time: string; // ISO timestamp
  question_count: number;
  starts_in_seconds: number;
}

// Данные для обратного отсчета (Тип: quiz:countdown)
export interface QuizCountdownData {
  quiz_id: number;
  seconds_left: number;
}

// --- НЕИСПОЛЬЗУЕМЫЕ/НЕИЗВЕСТНЫЕ ТИПЫ --- 
// Закомментированы типы, которые ожидал старый фронтенд или отправка которых бэкендом не подтверждена
/*
export interface QuizCountdownData { // Отправка не найдена на бэкенде
  quiz_id: number;
  seconds_left: number;
}
export interface QuizWaitingRoomData { // Отправка не найдена на бэкенде
  quiz_id: number;
  title: string;
  description: string;
  scheduled_time: string;
  question_count: number;
  starts_in_seconds: number;
}
export interface LeaderboardEntryData { // Отправка leaderboard целиком не найдена
  user_id: number;
  username: string;
  score: number;
  position: number;
  correct_answers: number;
}
export interface QuizLeaderboardData { // Отправка leaderboard целиком не найдена
  leaderboard: LeaderboardEntryData[];
}
// Типы токенов оставлены на случай использования, но отправка не найдена
export interface TokenExpireSoonData {
  expires_in_seconds: number;
}
export interface TokenExpiredData {
  message: string;
}
*/

// Объединенный тип для всех ИЗВЕСТНЫХ входящих сообщений от сервера
export type WsServerMessage =
  | { type: 'quiz:start'; data: QuizStartData }
  | { type: 'quiz:question'; data: QuizQuestionData }
  | { type: 'quiz:answer_result'; data: QuizAnswerResultData }
  | { type: 'quiz:timer'; data: QuizTimerData }
  | { type: 'quiz:answer_reveal'; data: QuizAnswerRevealData }
  | { type: 'quiz:elimination'; data: QuizEliminationData }
  | { type: 'quiz:elimination_reminder'; data: QuizEliminationReminderData }
  | { type: 'quiz:user_ready'; data: QuizUserReadyData }
  | { type: 'quiz:finish'; data: QuizFinishData }
  | { type: 'quiz:results_available'; data: QuizResultsAvailableData }
  | { type: 'server:heartbeat'; data: ServerHeartbeatData }
  | { type: 'error'; data: ErrorData }
  | { type: 'quiz:announcement'; data: QuizAnnouncementData }
  | { type: 'quiz:waiting_room'; data: QuizWaitingRoomData }
  | { type: 'quiz:countdown'; data: QuizCountdownData }
