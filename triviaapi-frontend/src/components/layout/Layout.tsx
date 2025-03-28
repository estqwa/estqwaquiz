import React, { ReactNode } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/router';
import { useAppSelector, useAppDispatch } from '../../hooks/redux-hooks';
import { logoutSuccess } from '../../store/auth/slice';
import { authService } from '../../api/services/authService';

interface LayoutProps {
  children: ReactNode;
  title?: string;
}

const Layout: React.FC<LayoutProps> = ({ children, title }) => {
  const router = useRouter();
  const dispatch = useAppDispatch();
  const { isAuthenticated, user } = useAppSelector(state => state.auth);

  const isActive = (path: string) => router.pathname === path || router.pathname.startsWith(`${path}/`);

  // Обработчик выхода из системы
  const handleLogout = async () => {
    try {
      await authService.logout();
      dispatch(logoutSuccess());
      router.push('/');
    } catch (error) {
      console.error('Error during logout:', error);
    }
  };

  return (
    <div className="min-h-screen bg-gray-50">
      <header className="bg-white shadow-sm">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between h-16">
            <div className="flex">
              <div className="flex-shrink-0 flex items-center">
                <Link href="/" className="text-xl font-bold text-blue-600">
                  Trivia App
                </Link>
              </div>
              <nav className="ml-6 flex space-x-8">
                <Link href="/" className={`inline-flex items-center px-1 pt-1 border-b-2 text-sm font-medium ${isActive('/') ? 'border-blue-500 text-gray-900' : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'}`}>
                  Главная
                </Link>
                <Link href="/play" className={`inline-flex items-center px-1 pt-1 border-b-2 text-sm font-medium ${isActive('/play') ? 'border-blue-500 text-gray-900' : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'}`}>
                  Играть
                </Link>
                {isAuthenticated && (
                  <>
                    <Link href="/my-quizzes" className={`inline-flex items-center px-1 pt-1 border-b-2 text-sm font-medium ${isActive('/my-quizzes') ? 'border-blue-500 text-gray-900' : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'}`}>
                      Мои викторины
                    </Link>
                    <Link href="/results" className={`inline-flex items-center px-1 pt-1 border-b-2 text-sm font-medium ${isActive('/results') ? 'border-blue-500 text-gray-900' : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'}`}>
                      Результаты
                    </Link>
                  </>
                )}
              </nav>
            </div>
            <div className="flex items-center">
              {isAuthenticated ? (
                <div className="flex items-center space-x-4">
                  <span className="text-sm text-gray-700">{user?.username}</span>
                  <Link href="/profile" className="text-sm text-blue-600 hover:text-blue-500">
                    Профиль
                  </Link>
                  <button 
                    onClick={handleLogout}
                    className="text-sm text-gray-700 hover:text-gray-500"
                  >
                    Выйти
                  </button>
                </div>
              ) : (
                <div className="flex items-center space-x-4">
                  <Link href="/auth/login" className="text-sm text-blue-600 hover:text-blue-500">
                    Войти
                  </Link>
                  <Link href="/auth/register" className="text-sm bg-blue-500 text-white px-3 py-1 rounded-md hover:bg-blue-600">
                    Регистрация
                  </Link>
                </div>
              )}
            </div>
          </div>
        </div>
      </header>

      <main className="py-6">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          {title && (
            <div className="mb-6">
              <h1 className="text-2xl font-bold text-gray-900">{title}</h1>
            </div>
          )}
          {children}
        </div>
      </main>

      <footer className="bg-white border-t border-gray-200 py-8">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center text-gray-500 text-sm">
            &copy; {new Date().getFullYear()} Trivia API. Все права защищены.
          </div>
        </div>
      </footer>
    </div>
  );
};

export default Layout; 