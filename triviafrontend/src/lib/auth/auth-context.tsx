"use client";

import { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { User, loginUser, registerUser, logoutUser, getCurrentUser, refreshTokens, RegisterRequest } from '../api/auth';
import { ApiError } from '../api/http-client';

// Интерфейс контекста аутентификации
interface AuthContextType {
  user: User | null;
  loading: boolean;
  error: string | null;
  login: (email: string, password: string) => Promise<void>;
  register: (data: RegisterRequest) => Promise<void>;
  logout: () => Promise<void>;
  refresh: () => Promise<void>;
  clearError: () => void;
  isAuthenticated: boolean;
  csrfToken: string | null;
}

// Создаем контекст
const AuthContext = createContext<AuthContextType | undefined>(undefined);

// Провайдер контекста
interface AuthProviderProps {
  children: ReactNode;
}

export function AuthProvider({ children }: AuthProviderProps) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [csrfToken, setCsrfToken] = useState<string | null>(null);

  // Эффект для первоначальной проверки аутентификации
  useEffect(() => {
    const checkAuthStatus = async () => {
      try {
        const userData = await getCurrentUser();
        setUser(userData);
        setIsAuthenticated(true);
        // Пробуем прочитать CSRF-токен из localStorage
        setCsrfToken(localStorage.getItem('csrf_token'));
      } catch (err) {
        // Если получаем 401, пробуем обновить токены
        if ((err as ApiError).status === 401) {
          try {
            await refreshTokens(null); // Передаем null, так как токен еще не установлен
            const userData = await getCurrentUser();
            setUser(userData);
            setIsAuthenticated(true);
            // После успешного обновления токенов пробуем прочитать CSRF-токен
            setCsrfToken(localStorage.getItem('csrf_token'));
          } catch (refreshErr) {
            // Если обновление не удалось, значит пользователь не авторизован
            setUser(null);
            setIsAuthenticated(false);
            setCsrfToken(null);
          }
        } else {
          setUser(null);
          setIsAuthenticated(false);
          setCsrfToken(null);
        }
      } finally {
        setLoading(false);
      }
    };

    checkAuthStatus();
  }, []);

  // Функция для входа
  const login = async (email: string, password: string) => {
    try {
      setLoading(true);
      setError(null);
      
      const response = await loginUser(email, password);
      setUser(response.user);
      setIsAuthenticated(true);
      
      // После успешного входа читаем CSRF токен из localStorage и устанавливаем в состояние
      const storedCsrfToken = localStorage.getItem('csrf_token');
      setCsrfToken(storedCsrfToken);
      console.log("login: CSRF Token прочитан из localStorage:", storedCsrfToken);
    } catch (err) {
      setError((err as ApiError).error || 'Ошибка входа в систему');
      throw err;
    } finally {
      setLoading(false);
    }
  };

  // Функция для регистрации
  const register = async (data: RegisterRequest) => {
    try {
      setLoading(true);
      setError(null);
      
      const response = await registerUser(data);
      setUser(response.user);
      setIsAuthenticated(true);
      
      // После успешной регистрации читаем CSRF токен из localStorage и устанавливаем в состояние
      const storedCsrfToken = localStorage.getItem('csrf_token');
      setCsrfToken(storedCsrfToken);
      console.log("register: CSRF Token прочитан из localStorage:", storedCsrfToken);
    } catch (err) {
      setError((err as ApiError).error || 'Ошибка регистрации');
      throw err;
    } finally {
      setLoading(false);
    }
  };

  // Функция для выхода
  const logout = async () => {
    try {
      setLoading(true);
      // Передаем текущий CSRF токен из состояния
      await logoutUser(csrfToken);
      
      // После успешного выхода очищаем состояние и localStorage
      setUser(null);
      setIsAuthenticated(false);
      setCsrfToken(null);
      localStorage.removeItem('csrf_token');
    } catch (err) {
      setError('Не удалось выйти из системы. Пожалуйста, попробуйте снова или обратитесь в поддержку.');
      throw err;
    } finally {
      setLoading(false);
    }
  };

  // Функция для обновления токенов
  const refresh = async () => {
    try {
      setLoading(true);
      // Передаем текущий CSRF токен из состояния
      await refreshTokens(csrfToken);
      
      // После обновления токенов получаем актуальные данные пользователя
      const userData = await getCurrentUser();
      setUser(userData);
      setIsAuthenticated(true);
      
      // Обновляем CSRF токен из localStorage (если он обновился на сервере)
      const storedCsrfToken = localStorage.getItem('csrf_token');
      setCsrfToken(storedCsrfToken);
      console.log("refresh: CSRF Token обновлен из localStorage:", storedCsrfToken);
    } catch (err) {
      setError((err as ApiError).error || 'Ошибка обновления токенов');
      setUser(null);
      setIsAuthenticated(false);
      setCsrfToken(null);
      localStorage.removeItem('csrf_token');
      throw err;
    } finally {
      setLoading(false);
    }
  };

  // Функция для очистки ошибок
  const clearError = () => {
    setError(null);
  };

  const value = {
    user,
    loading,
    error,
    login,
    register,
    logout,
    refresh,
    clearError,
    isAuthenticated,
    csrfToken,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

// Хук для использования контекста аутентификации
export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
} 