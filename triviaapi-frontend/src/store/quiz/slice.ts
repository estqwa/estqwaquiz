import { createSlice, PayloadAction } from '@reduxjs/toolkit';
import { Quiz } from '../../types/quiz';
import { Question } from '../../types/question';
import { UserAnswer } from '../../types/answer';
import { UserQuizResult } from '../../types/result';
import { QuizState } from './types';

// Начальное состояние
const initialState: QuizState = {
  activeQuiz: null,
  currentQuestion: null,
  questions: [],
  userAnswers: {},
  results: null,
  leaderboard: null,
  remainingTime: null,
  quizStatus: 'idle',
  isLoading: false,
  error: null,
  currentQuestionIndex: null,
  questionHistory: [],
  isSubmitting: false,
  hasSubmittedAnswer: false
};

const quizSlice = createSlice({
  name: 'quiz',
  initialState,
  reducers: {
    // Установка активной викторины
    setActiveQuiz: (state, action: PayloadAction<{ quiz: Quiz; questions?: Question[] }>) => {
      console.log('setActiveQuiz вызван с данными:', action.payload);
      state.activeQuiz = action.payload.quiz;
      state.questions = action.payload.questions || [];
      state.quizStatus = 'waiting';
      state.currentQuestion = null;
      state.userAnswers = {};
      state.results = null;
      state.leaderboard = null;
      state.error = null;
      state.isLoading = false;
      console.log('Новое состояние после setActiveQuiz:', { 
        activeQuiz: state.activeQuiz, 
        quizStatus: state.quizStatus 
      });
    },
    // Установка текущего вопроса
    setCurrentQuestion: (state, action: PayloadAction<Question>) => {
      console.log('setCurrentQuestion вызван с данными:', action.payload);
      // Добавляем вопрос в список, если его там еще нет
      if (!state.questions.some(q => q.id === action.payload.id)) {
        state.questions.push(action.payload);
      }
      state.currentQuestion = action.payload;
      state.quizStatus = 'active';
      state.remainingTime = action.payload.timeLimitSeconds;
      state.error = null;
      console.log('Новое состояние после setCurrentQuestion:', { 
        currentQuestion: state.currentQuestion, 
        quizStatus: state.quizStatus,
        remainingTime: state.remainingTime
      });
    },
    // Добавление/обновление ответа пользователя
    updateUserAnswer: (state, action: PayloadAction<UserAnswer>) => {
      state.userAnswers[action.payload.questionId] = action.payload;
    },
    // Обновление таймера вопроса
    updateRemainingTime: (state, action: PayloadAction<number>) => {
      console.log('updateRemainingTime вызван с данными:', action.payload);
      console.log('Текущий статус викторины:', state.quizStatus);
      
      // Обновляем таймер как для активной викторины, так и для режима ожидания
      // Это позволит отображать обратный отсчет перед началом викторины
      if (state.quizStatus === 'active' || state.quizStatus === 'waiting') {
        state.remainingTime = Math.max(0, action.payload);
        console.log('Обновлено remainingTime:', state.remainingTime);
      } else {
        console.log('Не обновляем таймер, так как статус не active и не waiting');
      }
    },
    // Завершение вопроса
    endCurrentQuestion: (state, action: PayloadAction<{ questionId: number; correctAnswer?: number }>) => {
      if (state.currentQuestion?.id === action.payload.questionId) {
        state.currentQuestion = null;
        state.remainingTime = 0;
      }
    },
    // Завершение викторины
    setQuizEnded: (state) => {
      state.quizStatus = 'ended';
      state.currentQuestion = null;
      state.remainingTime = null;
    },
    // Отмена викторины
    setQuizCancelled: (state) => {
      state.quizStatus = 'cancelled';
      state.activeQuiz = null;
      state.currentQuestion = null;
      state.remainingTime = null;
      state.userAnswers = {};
    },
    // Установка финальных результатов пользователя
    setUserResults: (state, action: PayloadAction<UserQuizResult>) => {
      state.results = action.payload;
    },
    // Установка таблицы лидеров
    setLeaderboard: (state, action: PayloadAction<UserQuizResult[]>) => {
      state.leaderboard = action.payload;
    },
    // Сброс состояния викторины
    resetQuizState: (state) => {
      return initialState;
    },
    // Установка ошибки викторины
    setQuizError: (state, action: PayloadAction<string>) => {
      state.error = action.payload;
      state.isLoading = false;
    },
    // Начало загрузки данных викторины
    quizLoadingStart: (state) => {
      state.isLoading = true;
      state.error = null;
    }
  },
});

export const {
  setActiveQuiz,
  setCurrentQuestion,
  updateUserAnswer,
  updateRemainingTime,
  endCurrentQuestion,
  setQuizEnded,
  setQuizCancelled,
  setUserResults,
  setLeaderboard,
  resetQuizState,
  setQuizError,
  quizLoadingStart
} = quizSlice.actions;

export default quizSlice.reducer; 