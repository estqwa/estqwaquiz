"use client";

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { useAuth } from '../lib/auth/auth-context';

export default function Navbar() {
  const pathname = usePathname();
  const { user, isAuthenticated, logout } = useAuth();
  
  const isActive = (path: string) => {
    return pathname === path ? 'text-blue-500 border-b-2 border-blue-500' : 'text-gray-700 hover:text-blue-500';
  };
  
  const handleLogout = async () => {
    try {
      await logout();
    } catch (error) {
      console.error('Ошибка при выходе из системы:', error);
      alert('Ошибка выхода: ' + (error as Error).message);
    }
  };
  
  return (
    <nav className="bg-white shadow-md py-4">
      <div className="container mx-auto px-4">
        <div className="flex justify-between items-center">
          <div className="text-xl font-bold text-blue-600">
            <Link href="/">Trivia API</Link>
          </div>
          <div className="space-x-6">
            <Link href="/" className={`${isActive('/')} px-2 py-1`}>
              Главная
            </Link>
            
            {isAuthenticated ? (
              <>
                <Link href="/quizzes" className={`${isActive('/quizzes')} px-2 py-1`}>
                  Викторины
                </Link>
                <Link href="/results" className={`${isActive('/results')} px-2 py-1`}>
                  Результаты
                </Link>
                <span className="text-gray-600 px-2 py-1">
                  {user?.username}
                </span>
                <button 
                  onClick={handleLogout}
                  className="text-red-500 hover:text-red-700 px-2 py-1"
                >
                  Выйти
                </button>
              </>
            ) : (
              <>
                <Link href="/login" className={`${isActive('/login')} px-2 py-1`}>
                  Вход
                </Link>
                <Link href="/register" className={`${isActive('/register')} px-2 py-1`}>
                  Регистрация
                </Link>
              </>
            )}
          </div>
        </div>
      </div>
    </nav>
  );
} 