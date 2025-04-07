"use client";

import { ReactNode } from 'react';
import ProtectedRoute from '../../../../components/ProtectedRoute';

interface LiveQuizLayoutProps {
  children: ReactNode;
}

/**
 * Layout для страниц активных викторин
 * Требует аутентификации пользователя
 */
export default function LiveQuizLayout({ children }: LiveQuizLayoutProps) {
  return (
    <ProtectedRoute>
      <div className="max-w-5xl mx-auto">
        {children}
      </div>
    </ProtectedRoute>
  );
} 