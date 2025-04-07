"use client";

import { ReactNode } from 'react';
import ProtectedRoute from '../../components/ProtectedRoute';

interface QuizLayoutProps {
  children: ReactNode;
}

/**
 * Layout для страниц отдельных викторин
 * Требует аутентификации пользователя
 */
export default function QuizLayout({ children }: QuizLayoutProps) {
  return (
    <ProtectedRoute>
      <div className="max-w-5xl mx-auto">
        {children}
      </div>
    </ProtectedRoute>
  );
} 