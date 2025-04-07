"use client";

import { ReactNode } from 'react';
import ProtectedRoute from '../../components/ProtectedRoute';

interface QuizzesLayoutProps {
  children: ReactNode;
}

/**
 * Layout для страниц викторин
 * Все страницы викторин требуют аутентификации
 */
export default function QuizzesLayout({ children }: QuizzesLayoutProps) {
  return (
    <ProtectedRoute>
      <div className="max-w-5xl mx-auto">
        {children}
      </div>
    </ProtectedRoute>
  );
} 