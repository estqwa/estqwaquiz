import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { authService, LoginRequest, RegisterRequest } from '../../api/services/authService';
import { User } from '../../types/user';
import { useDispatch } from 'react-redux';
import { 
  authRequestStart,
  authSuccess, 
  authFailure, 
  logoutSuccess 
} from '../../store/auth/slice';
import React from 'react';
import { store } from '../../store';

// Ключи запросов для React Query
const userKeys = {
  all: ['users'] as const,
  current: () => [...userKeys.all, 'current'] as const,
  profile: (id: number) => [...userKeys.all, 'profile', id] as const,
};

/**
 * Хук для получения данных текущего пользователя
 */
export const useCurrentUser = () => {
  const dispatch = useDispatch();

  const result = useQuery({
    queryKey: userKeys.current(),
    queryFn: async () => {
      try {
        const user = await authService.getCurrentUser();
        // Обновляем состояние Redux при успешном получении пользователя
        dispatch(authSuccess({ user }));
        return user;
      } catch (error) {
        // Если пользователь не аутентифицирован, не выбрасываем ошибку, а сбрасываем состояние
        dispatch(logoutSuccess());
        throw error;
      }
    },
    staleTime: 5 * 60 * 1000, // 5 минут - данные считаются свежими
    retry: 1, // Повторять запрос 1 раз при ошибке
  });

  // Обрабатываем успех и ошибку вне объекта useQuery
  React.useEffect(() => {
    if (result.isSuccess && result.data) {
      dispatch(authSuccess({ user: result.data }));
    } else if (result.isError) {
      dispatch(logoutSuccess());
    }
  }, [result.isSuccess, result.isError, result.data, dispatch]);

  return result;
};

/**
 * Хук для входа пользователя в систему
 */
export const useLogin = () => {
  const dispatch = useDispatch();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (credentials: LoginRequest) => {
      dispatch(authRequestStart());
      return authService.login(credentials);
    },
    onSuccess: (data) => {
      // Проверяем, содержит ли ответ пользователя
      if (!data || !data.user) {
        console.error('Invalid response format: missing user data', data);
        dispatch(authFailure('Некорректный формат ответа сервера'));
        return;
      }

      const { useCookieAuth } = store.getState().auth;
      
      // Обновляем состояние Redux при успешном входе
      dispatch(authSuccess({
        user: data.user,
        // В режиме Cookie-based auth, access_token хранится в HttpOnly cookie
        // и не должен сохраняться в redux state
        token: !useCookieAuth && data.access_token ? data.access_token : undefined,
        csrfToken: data.csrf_token
      }));

      // Если используется Bearer Auth, сохраняем токены в localStorage
      if (!useCookieAuth && data.access_token) {
        localStorage.setItem('access_token', data.access_token);
        if (data.refresh_token) {
          localStorage.setItem('refresh_token', data.refresh_token);
        }
      }

      // Обновляем данные пользователя в кэше React Query
      queryClient.setQueryData(userKeys.current(), data.user);
    },
    onError: (error: any) => {
      // Подробное логирование ошибки для диагностики
      console.error('Login error:', error);
      console.log('Response data:', error.response?.data);
      console.log('Response status:', error.response?.status);
      
      const errorMessage = error.response?.data?.error?.message || 
                          error.response?.data?.message || 
                          error.message || 
                          'Login failed';
      
      dispatch(authFailure(errorMessage));
    },
  });
};

/**
 * Хук для регистрации нового пользователя
 */
export const useRegister = () => {
  const dispatch = useDispatch();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (userData: RegisterRequest) => {
      dispatch(authRequestStart());
      return authService.register(userData);
    },
    onSuccess: (data) => {
      // Обновляем состояние Redux при успешной регистрации
      dispatch(authSuccess({
        user: data.user,
        token: data.access_token,
        csrfToken: data.csrf_token
      }));

      // Если используется Bearer Auth, сохраняем токены в localStorage
      if (data.access_token) {
        localStorage.setItem('access_token', data.access_token);
        if (data.refresh_token) {
          localStorage.setItem('refresh_token', data.refresh_token);
        }
      }

      // Обновляем данные пользователя в кэше React Query
      queryClient.setQueryData(userKeys.current(), data.user);
    },
    onError: (error: any) => {
      const errorMessage = error.response?.data?.error?.message || 'Registration failed';
      dispatch(authFailure(errorMessage));
    },
  });
};

/**
 * Хук для выхода из системы
 */
export const useLogout = () => {
  const dispatch = useDispatch();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => authService.logout(),
    onSuccess: () => {
      // Очищаем состояние Redux при успешном выходе
      dispatch(logoutSuccess());
      
      // Инвалидируем кэш React Query для данных пользователя
      queryClient.invalidateQueries({ queryKey: userKeys.current() });
      queryClient.removeQueries({ queryKey: userKeys.current() });
    },
    onError: () => {
      // Даже при ошибке на сервере, выполняем локальный выход
      dispatch(logoutSuccess());
      
      // Инвалидируем кэш React Query для данных пользователя
      queryClient.invalidateQueries({ queryKey: userKeys.current() });
      queryClient.removeQueries({ queryKey: userKeys.current() });
    },
  });
};

/**
 * Хук для проверки аутентификации при инициализации приложения
 */
export const useCheckAuth = () => {
  const dispatch = useDispatch();

  return useMutation({
    mutationFn: () => authService.checkAuth(),
    onSuccess: (isAuthenticated) => {
      if (isAuthenticated) {
        // Если пользователь аутентифицирован, обновляем данные
        authService.getCurrentUser()
          .then(user => {
            dispatch(authSuccess({ user }));
          })
          .catch(() => {
            dispatch(logoutSuccess());
          });
      } else {
        dispatch(logoutSuccess());
      }
    },
    onError: () => {
      dispatch(logoutSuccess());
    },
  });
}; 