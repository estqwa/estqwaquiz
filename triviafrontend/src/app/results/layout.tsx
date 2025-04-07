"use client";

import { ReactNode } from 'react';
import ProtectedRoute from '../../components/ProtectedRoute';

interface ResultsLayoutProps {
  children: ReactNode;
}

/**
 * Layout для страниц результатов
 * Доступ к результатам требует аутентификации
 */
export default function ResultsLayout({ children }: ResultsLayoutProps) {
  return (
    <ProtectedRoute>
      <div className="max-w-5xl mx-auto">
        {children}
      </div>
    </ProtectedRoute>
  );
} 