import { createSlice, PayloadAction } from '@reduxjs/toolkit';
import { User } from '../../types/user';
import { AuthState } from './types';

// Начальное состояние
const initialState: AuthState = {
  user: null,
  csrfToken: null, // CSRF Token (для Cookie Auth)
  isAuthenticated: false,
  isLoading: false,
  error: null,
  status: 'idle',
  expiresAt: null,
  activeSessions: []
};

const authSlice = createSlice({
  name: 'auth',
  initialState,
  reducers: {
    // Экшен при начале запроса на логин/регистрацию
    authRequestStart: (state) => {
      state.isLoading = true;
      state.error = null;
      state.status = 'loading';
    },
    
    // Экшен при успешном логине/регистрации/получении пользователя
    loginSuccess: (state, action: PayloadAction<{ user: User | null; csrfToken: string | null }>) => {
      // Проверяем наличие пользователя
      if (!action.payload.user) {
        console.error('loginSuccess called without user data:', action.payload);
        return;
      }
      
      state.isLoading = false;
      state.isAuthenticated = true;
      state.user = action.payload.user;
      state.csrfToken = action.payload.csrfToken || null;
      state.error = null;
      state.status = 'succeeded';
    },
    
    // Экшен при ошибке логина/регистрации
    authFailure: (state, action: PayloadAction<string>) => {
      state.isLoading = false;
      state.isAuthenticated = false;
      state.user = null;
      state.csrfToken = null;
      state.error = action.payload;
      state.status = 'failed';
    },
    
    // Экшен при успешном выходе
    logoutSuccess: (state) => {
      state.user = null;
      state.csrfToken = null;
      state.isAuthenticated = false;
      state.isLoading = false;
      state.error = null;
      state.status = 'idle';
    },
    
    // Экшен для установки пользователя при проверке сессии
    setUser: (state, action: PayloadAction<User | null>) => {
      state.user = action.payload;
      state.isAuthenticated = !!action.payload;
      state.isLoading = false;
      state.status = 'succeeded';
    },
    
    // Экшен для обновления данных пользователя
    updateUser: (state, action: PayloadAction<Partial<User>>) => {
      if (state.user) {
        state.user = { ...state.user, ...action.payload };
      }
    },
    
    // Очистка ошибки аутентификации
    clearAuthError: (state) => {
      state.error = null;
    },
    
    // Экшен для установки статуса проверки аутентификации
    setAuthStatus: (state, action: PayloadAction<'idle' | 'loading' | 'succeeded' | 'failed' | 'checked'>) => {
      state.status = action.payload;
    },
    
    // Экшен для установки флага завершения проверки аутентификации
    setAuthChecked: (state) => {
      state.status = 'checked';
    },
    
    // Экшен для обновления CSRF-токена (если используется)
    updateCsrfToken: (state, action: PayloadAction<string | null>) => {
      state.csrfToken = action.payload;
    }
  },
});

export const {
  authRequestStart,
  loginSuccess,
  authFailure,
  logoutSuccess,
  setUser,
  updateUser,
  clearAuthError,
  setAuthStatus,
  setAuthChecked,
  updateCsrfToken
} = authSlice.actions;

export default authSlice.reducer; 