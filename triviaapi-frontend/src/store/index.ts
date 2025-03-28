import { configureStore } from '@reduxjs/toolkit';
import authReducer from './auth/slice';
import quizReducer from './quiz/slice';
import websocketReducer from './websocket/slice';

export const store = configureStore({
  reducer: {
    auth: authReducer,
    quiz: quizReducer,
    websocket: websocketReducer,
  },
});

// Типы для использования с React-Redux хуками
export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch; 