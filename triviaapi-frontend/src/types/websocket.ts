/**
 * Интерфейс WebSocket сообщения общего формата
 */
export interface WebSocketMessage {
  type: string; // Тип события
  data: unknown; // Данные сообщения, зависящие от типа
  priority?: MessagePriority; // Приоритет сообщения
}

/**
 * Типы WebSocket событий
 * Названия констант должны точно соответствовать именам на бэкенде
 */
export enum WebSocketEventType {
  // События аутентификации
  TOKEN_EXPIRE_SOON = 'TOKEN_EXPIRE_SOON',
  TOKEN_EXPIRED = 'TOKEN_EXPIRED',
  TOKEN_REVOKED = 'TOKEN_REVOKED',
  TOKEN_INVALIDATED = 'TOKEN_INVALIDATED',
  TOKEN_REFRESHED = 'TOKEN_REFRESHED',
  
  // События викторины
  QUIZ_START = 'QUIZ_START',
  QUIZ_END = 'QUIZ_END',
  QUESTION_START = 'QUESTION_START',
  QUESTION_END = 'QUESTION_END',
  USER_ANSWER = 'USER_ANSWER',
  RESULT_UPDATE = 'RESULT_UPDATE',
  
  // Системные события
  USER_HEARTBEAT = 'user:heartbeat',
  SERVER_HEARTBEAT = 'server:heartbeat',
  
  // Дополнительные события (проверить соответствие на бэкенде)
  QUIZ_TIMER = 'QUIZ_TIMER',
  QUIZ_CANCELLED = 'QUIZ_CANCELLED',
  QUIZ_ANNOUNCEMENT = 'QUIZ_ANNOUNCEMENT',
  QUIZ_WAITING_ROOM = 'QUIZ_WAITING_ROOM',
  QUIZ_COUNTDOWN = 'QUIZ_COUNTDOWN',
  QUIZ_ANSWER_REVEAL = 'QUIZ_ANSWER_REVEAL',
  QUIZ_ANSWER_RESULT = 'QUIZ_ANSWER_RESULT',
  QUIZ_LEADERBOARD = 'QUIZ_LEADERBOARD',
  QUIZ_USER_READY = 'QUIZ_USER_READY'
}

/**
 * Приоритеты сообщений
 */
export enum MessagePriority {
  CRITICAL = 3,
  HIGH = 2,
  NORMAL = 1,
  LOW = 0
}

/**
 * События связанные с токенами
 */
export interface TokenRefreshedEvent {
  user_id: number;
  device_id?: string;
  access_token: string;
  csrf_token: string;
  expires_in: number;
}

export interface TokenInvalidatedEvent {
  user_id: number;
  device_id?: string;
  token_id?: string;
  reason: string;
}

export interface TokenExpiryWarningEvent {
  user_id: number;
  expires_in: number; // секунды до истечения
  token_id?: string;
}

export interface KeyRotationEvent {
  user_id: number;
  device_id?: string;
  access_token: string;
  csrf_token: string;
  expires_in: number;
  rotation_reason?: string; // Причина ротации (плановая, внеплановая и т.д.)
}

/**
 * События викторин
 */
export interface QuizStartEvent {
  quiz_id: number;
  title: string;
  description: string;
  num_questions: number;
  duration_minutes: number;
  start_time: string; // ISO 8601 формат даты
}

export interface QuizEndEvent {
  quiz_id: number;
  message: string;
  winners?: Array<{
    user_id: number;
    username: string;
    score: number;
    position: number;
  }>;
}

export interface QuestionStartEvent {
  quiz_id: number;
  question_id: number;
  question_number: number;
  text: string;
  options: Array<{
    id: number;
    text: string;
  }>;
  duration_seconds: number;
  total_questions: number;
  start_time: string; // ISO 8601 формат даты
}

export interface QuestionEndEvent {
  quiz_id: number;
  question_id: number;
  correct_option_id: number;
  explanation?: string;
}

export interface UserAnswerEvent {
  quiz_id: number;
  question_id: number;
  option_id: number;
  answer_time: string; // ISO 8601 формат даты
}

export interface ResultUpdateEvent {
  quiz_id: number;
  leaderboard: Array<{
    user_id: number;
    username: string;
    score: number;
    position: number;
  }>;
  user_stats?: {
    correct_answers: number;
    total_answers: number;
    position: number;
    score: number;
  };
}

export interface QuizTimerEvent {
  quiz_id: number;
  question_id: number;
  remaining_seconds: number;
}

export interface QuizCancelledEvent {
  quiz_id: number;
  message: string;
  reason?: string;
}

/**
 * События анонса и ожидания викторины
 */
export interface QuizAnnouncementEvent {
  quiz_id: number;
  title: string;
  description: string;
  scheduled_time: string;
  question_count: number;
}

export interface QuizWaitingRoomEvent {
  quiz_id: number;
  title: string;
  starts_in_seconds: number;
}

export interface QuizCountdownEvent {
  quiz_id: number;
  seconds_left: number;
}

export interface QuizAnswerRevealEvent {
  question_id: number;
  correct_option: number;
}

export interface QuizAnswerResultEvent {
  question_id: number;
  correct_option: number;
  your_answer: number;
  is_correct: boolean;
  points_earned: number;
  time_taken_ms: number;
}

export interface QuizLeaderboardEvent {
  quiz_id: number;
  results: Array<{
    user_id: number;
    username: string;
    score: number;
    correct_answers: number;
    rank: number;
  }>;
}

export interface QuizUserReadyEvent {
  user_id: number;
  quiz_id: number;
  status: string;
}

/**
 * Системные события
 */
export interface ShardMigrationEvent {
  old_shard_id: number;
  new_shard_id: number;
  migration_token: string;
  migration_reason: string;
} 