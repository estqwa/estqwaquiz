"use client";

import { ReactNode, useEffect } from 'react';
import { useAuth } from '../lib/auth/auth-context';
import { redirect } from 'next/navigation';

interface ProtectedRouteProps {
  children: ReactNode;
  redirectTo?: string;
}

/**
 * Компонент для защиты маршрутов, требующих аутентификации
 * Если пользователь не аутентифицирован, происходит перенаправление на страницу входа
 */
export default function ProtectedRoute({ 
  children, 
  redirectTo = '/login' 
}: ProtectedRouteProps) {
  const { isAuthenticated, loading } = useAuth();

  useEffect(() => {
    // Проверяем только когда loading = false (завершена проверка аутентификации)
    if (!loading && !isAuthenticated) {
      redirect(redirectTo);
    }
  }, [loading, isAuthenticated, redirectTo]);

  // Показываем индикатор загрузки или ничего, пока проверяется состояние аутентификации
  if (loading) {
    return (
      <div className="flex justify-center items-center min-h-[50vh]">
        <div className="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-blue-500"></div>
      </div>
    );
  }

  // Если аутентифицирован, показываем содержимое
  if (isAuthenticated) {
    return <>{children}</>;
  }

  // Этот блок обычно не выполняется, т.к. редирект происходит в useEffect
  // Но оставляем его для безопасности
  return null;
} 