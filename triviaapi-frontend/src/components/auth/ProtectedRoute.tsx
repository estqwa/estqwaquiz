import React, { useEffect } from 'react';
import { useRouter } from 'next/router';
import { useAppSelector } from '../../hooks/redux-hooks';
import { useCurrentUser } from '../../hooks/query-hooks/useUserQueries';

interface ProtectedRouteProps {
  children: React.ReactNode;
  fallbackUrl?: string; // URL для перенаправления при отсутствии аутентификации
}

/**
 * Компонент для защиты маршрутов, требующих аутентификации.
 * Проверяет аутентификацию пользователя и перенаправляет на страницу входа,
 * если пользователь не аутентифицирован.
 */
const ProtectedRoute: React.FC<ProtectedRouteProps> = ({ 
  children, 
  fallbackUrl = '/auth/login' 
}) => {
  const router = useRouter();
  const { isAuthenticated, isLoading } = useAppSelector(state => state.auth);
  
  // Используем React Query для получения данных пользователя, если isAuthenticated
  const { refetch, isLoading: isUserLoading } = useCurrentUser();

  useEffect(() => {
    // Если пользователь не аутентифицирован и загрузка завершена, 
    // перенаправляем на страницу входа
    if (!isAuthenticated && !isLoading) {
      router.push(fallbackUrl);
    }
    
    // Если пользователь аутентифицирован, проверяем наличие данных
    if (isAuthenticated) {
      refetch();
    }
  }, [isAuthenticated, isLoading, router, fallbackUrl, refetch]);

  // Показываем загрузку, пока проверяем аутентификацию
  if (isLoading || isUserLoading) {
    return <div>Loading...</div>;
  }

  // Если пользователь аутентифицирован, возвращаем дочерние компоненты
  return isAuthenticated ? <>{children}</> : null;
};

export default ProtectedRoute; 