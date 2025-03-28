import { createSlice, PayloadAction } from '@reduxjs/toolkit';
import { User } from '../../types/user';
import { AuthState } from './types';

// Начальное состояние
const initialState: AuthState = {
  user: null,
  token: null, // Access Token (для Bearer)
  csrfToken: null, // CSRF Token (для Cookie Auth)
  isAuthenticated: false,
  isLoading: false,
  error: null,
  useCookieAuth: true, // По умолчанию используем Cookie Auth как более безопасный
  refreshToken: null,
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
    },
    // Экшен при успешном логине/регистрации/получении пользователя
    authSuccess: (state, action: PayloadAction<{ user: User; token?: string; csrfToken?: string }>) => {
      // Проверяем наличие пользователя
      if (!action.payload.user) {
        console.error('authSuccess called without user data:', action.payload);
        return;
      }
      
      state.isLoading = false;
      state.isAuthenticated = true;
      state.user = action.payload.user;
      state.token = action.payload.token || null;
      state.csrfToken = action.payload.csrfToken || null;
      state.error = null;
    },
    // Экшен при ошибке логина/регистрации
    authFailure: (state, action: PayloadAction<string>) => {
      state.isLoading = false;
      state.isAuthenticated = false;
      state.user = null;
      state.token = null;
      state.csrfToken = null;
      state.error = action.payload;
    },
    // Экшен при успешном выходе
    logoutSuccess: (state) => {
      state.user = null;
      state.token = null;
      state.csrfToken = null;
      state.isAuthenticated = false;
      state.isLoading = false;
      state.error = null;
    },
    // Экшен при успешном обновлении токенов
    tokenRefreshed: (state, action: PayloadAction<{ token?: string | null; csrfToken?: string | null }>) => {
      if (action.payload.token !== undefined) {
        state.token = action.payload.token;
      }
      if (action.payload.csrfToken !== undefined) {
        state.csrfToken = action.payload.csrfToken;
      }
      state.isLoading = false;
      state.isAuthenticated = true;
    },
    // Экшен для обновления данных пользователя
    updateUser: (state, action: PayloadAction<Partial<User>>) => {
      if (state.user) {
        state.user = { ...state.user, ...action.payload };
      }
    },
    // Экшен для переключения режима аутентификации
    setAuthMode: (state, action: PayloadAction<'cookie' | 'bearer'>) => {
      state.useCookieAuth = action.payload === 'cookie';
      // При смене режима сбрасываем токены другого режима
      if (state.useCookieAuth) {
        state.token = null;
      } else {
        state.csrfToken = null;
      }
    },
    // Очистка ошибки аутентификации
    clearAuthError: (state) => {
      state.error = null;
    }
  },
});

export const {
  authRequestStart,
  authSuccess,
  authFailure,
  logoutSuccess,
  tokenRefreshed,
  updateUser,
  setAuthMode,
  clearAuthError,
} = authSlice.actions;

export default authSlice.reducer; 