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
  QUIZ_START = 'quiz:start',
  QUIZ_END = 'quiz:end',
  QUESTION_START = 'quiz:question',       // На бэкенде соответствует quiz:question
  QUESTION_END = 'QUESTION_END',          // На бэкенде используется QUESTION_END
  USER_ANSWER = 'USER_ANSWER',            // На бэкенде используется USER_ANSWER
  RESULT_UPDATE = 'RESULT_UPDATE',        // На бэкенде используется RESULT_UPDATE
  
  // Системные события
  USER_HEARTBEAT = 'user:heartbeat',
  SERVER_HEARTBEAT = 'server:heartbeat',
  
  // Дополнительные события
  QUIZ_TIMER = 'quiz:timer',
  QUIZ_CANCELLED = 'quiz:cancelled',
  QUIZ_ANNOUNCEMENT = 'quiz:announcement',
  QUIZ_WAITING_ROOM = 'quiz:waiting_room',
  QUIZ_COUNTDOWN = 'quiz:countdown',
  QUIZ_ANSWER_REVEAL = 'quiz:answer_reveal',
  QUIZ_ANSWER_RESULT = 'quiz:answer_result',
  QUIZ_LEADERBOARD = 'quiz:leaderboard',
  QUIZ_USER_READY = 'quiz:user_ready',
  QUIZ_RESULTS_AVAILABLE = 'quiz:results_available',
  QUIZ_FINISH = 'quiz:finish'              // Дополнительно обнаружено в коде
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
  userId: number;
  deviceId?: string;
  accessToken: string;
  csrfToken: string;
  expiresIn: number;
}

export interface TokenInvalidatedEvent {
  userId: number;
  deviceId?: string;
  tokenId?: string;
  reason: string;
}

export interface TokenExpiryWarningEvent {
  userId: number;
  expiresIn: number; // секунды до истечения
  tokenId?: string;
}

export interface KeyRotationEvent {
  userId: number;
  deviceId?: string;
  accessToken: string;
  csrfToken: string;
  expiresIn: number;
  rotationReason?: string; // Причина ротации (плановая, внеплановая и т.д.)
}

/**
 * События викторин
 */
export interface QuizStartEvent {
  quizId: number;
  title: string;
  description: string;
  numQuestions: number;
  durationMinutes: number;
  startTime: string; // ISO 8601 формат даты
}

export interface QuizEndEvent {
  quizId: number;
  message: string;
  winners?: Array<{
    userId: number;
    username: string;
    score: number;
    position: number;
  }>;
}

export interface QuestionStartEvent {
  quizId: number;
  questionId: number;
  questionNumber: number;
  text: string;
  options: Array<{
    id: number;
    text: string;
  }>;
  durationSeconds: number;
  totalQuestions: number;
  startTime: string; // ISO 8601 формат даты
}

export interface QuestionEndEvent {
  quizId: number;
  questionId: number;
  correctOptionId: number;
  explanation?: string;
}

export interface UserAnswerEvent {
  quizId: number;
  questionId: number;
  optionId: number;
  answerTime: string; // ISO 8601 формат даты
}

export interface ResultUpdateEvent {
  quizId: number;
  leaderboard: Array<{
    userId: number;
    username: string;
    score: number;
    position: number;
  }>;
  userStats?: {
    correctAnswers: number;
    totalAnswers: number;
    position: number;
    score: number;
  };
}

export interface QuizTimerEvent {
  quizId: number;
  questionId: number;
  remainingSeconds: number;
}

export interface QuizCancelledEvent {
  quizId: number;
  message: string;
  reason?: string;
}

/**
 * События анонса и ожидания викторины
 */
export interface QuizAnnouncementEvent {
  quizId: number;
  title: string;
  description: string;
  scheduledTime: string;
  questionCount: number;
}

export interface QuizWaitingRoomEvent {
  quizId: number;
  title: string;
  startsInSeconds: number;
}

export interface QuizCountdownEvent {
  quizId: number;
  secondsLeft: number;
}

export interface QuizAnswerRevealEvent {
  questionId: number;
  correctOption: number;
}

export interface QuizAnswerResultEvent {
  questionId: number;
  correctOption: number;
  yourAnswer: number;
  isCorrect: boolean;
  pointsEarned: number;
  timeTakenMs: number;
}

export interface QuizLeaderboardEvent {
  quizId: number;
  results: Array<{
    userId: number;
    username: string;
    score: number;
    correctAnswers: number;
    rank: number;
  }>;
}

export interface QuizUserReadyEvent {
  userId: number;
  quizId: number;
  status: string;
}

/**
 * Системные события
 */
export interface ShardMigrationEvent {
  oldShardId: number;
  newShardId: number;
  migrationToken: string;
  migrationReason: string;
} 