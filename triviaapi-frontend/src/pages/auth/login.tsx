import React, { useEffect, useState } from 'react';
import { useRouter } from 'next/router';
import { useLogin } from '../../hooks/query-hooks/useUserQueries';
import { useAppSelector } from '../../hooks/redux-hooks';
import Link from 'next/link';

const LoginPage: React.FC = () => {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [tokenBased, setTokenBased] = useState(false);

  const router = useRouter();
  const { isAuthenticated, isLoading, error } = useAppSelector(state => state.auth);
  
  // Используем хук для входа
  const loginMutation = useLogin();

  useEffect(() => {
    // Если пользователь уже аутентифицирован, перенаправляем на главную страницу
    if (isAuthenticated) {
      router.push('/');
    }
  }, [isAuthenticated, router]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!email || !password) {
      return;
    }

    try {
      // Вызываем мутацию для входа
      loginMutation.mutate({ 
        email, 
        password, 
        token_based: tokenBased 
      });
    } catch (error) {
      console.error('Error in handleSubmit:', error);
    }
  };

  // Если идет процесс аутентификации или уже аутентифицированы, показываем индикатор загрузки
  if (isLoading || isAuthenticated) {
    return <div className="flex items-center justify-center min-h-screen">Loading...</div>;
  }

  return (
    <div className="flex items-center justify-center min-h-screen bg-gray-100">
      <div className="w-full max-w-md p-8 space-y-8 bg-white rounded-lg shadow-md">
        <div className="text-center">
          <h1 className="text-2xl font-bold">Вход в систему</h1>
          <p className="mt-2 text-gray-600">Войдите в свою учетную запись</p>
        </div>

        {error && (
          <div className="p-4 mb-4 text-sm text-red-700 bg-red-100 rounded-lg" role="alert">
            <div className="font-bold">Ошибка при входе:</div>
            <div>{error}</div>
            {loginMutation.error && (
              <div className="mt-1 text-xs">
                Детали: {loginMutation.error.message || 'Нет дополнительной информации'}
              </div>
            )}
          </div>
        )}

        <form className="mt-8 space-y-6" onSubmit={handleSubmit}>
          <div>
            <label htmlFor="email" className="block text-sm font-medium text-gray-700">
              Email
            </label>
            <input
              id="email"
              name="email"
              type="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="block w-full px-3 py-2 mt-1 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500"
            />
          </div>

          <div>
            <label htmlFor="password" className="block text-sm font-medium text-gray-700">
              Пароль
            </label>
            <input
              id="password"
              name="password"
              type="password"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="block w-full px-3 py-2 mt-1 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500"
            />
          </div>

          <div className="flex items-center">
            <input
              id="token-based"
              name="token-based"
              type="checkbox"
              checked={tokenBased}
              onChange={(e) => setTokenBased(e.target.checked)}
              className="w-4 h-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500"
            />
            <label htmlFor="token-based" className="block ml-2 text-sm text-gray-900">
              Использовать Bearer Token аутентификацию
            </label>
          </div>

          <div>
            <button
              type="submit"
              disabled={loginMutation.isPending}
              className="flex justify-center w-full px-4 py-2 text-sm font-medium text-white bg-indigo-600 border border-transparent rounded-md shadow-sm hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50"
            >
              {loginMutation.isPending ? 'Выполняется вход...' : 'Войти'}
            </button>
          </div>
        </form>

        <div className="text-center mt-4">
          <p className="text-sm text-gray-600">
            Нет учетной записи?{' '}
            <Link href="/auth/register">
              <span className="font-medium text-indigo-600 hover:text-indigo-500 cursor-pointer">
                Зарегистрируйтесь
              </span>
            </Link>
          </p>
        </div>
      </div>
    </div>
  );
};

export default LoginPage; 