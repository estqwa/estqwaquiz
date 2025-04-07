"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useAuth } from "../lib/auth/auth-context";
import { Quiz, getScheduledQuizzes } from "../lib/api/quizzes";
import { formatDate, DateFormat, compareDates } from "../lib/utils/dateUtils";

export default function Home() {
  const { isAuthenticated, user } = useAuth();
  const [scheduledQuizzes, setScheduledQuizzes] = useState<Quiz[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (isAuthenticated) {
      const fetchScheduledQuizzes = async () => {
        try {
          setLoading(true);
          const quizzes = await getScheduledQuizzes();
          setScheduledQuizzes(quizzes);
        } catch (err) {
          setError("Не удалось загрузить запланированные викторины");
          console.error(err);
        } finally {
          setLoading(false);
        }
      };

      fetchScheduledQuizzes();
    }
  }, [isAuthenticated]);

  // Находим ближайшую запланированную викторину
  const upcomingQuiz = scheduledQuizzes.length > 0 
    ? scheduledQuizzes.sort((a, b) => 
        compareDates(a.scheduled_time, b.scheduled_time)
      )[0] 
    : null;

  return (
    <div className="flex flex-col items-center justify-center">
      <div className="text-center max-w-3xl mx-auto mb-12">
        <h1 className="text-4xl font-bold mb-4">Добро пожаловать в Trivia API</h1>
        <p className="text-xl text-gray-600 mb-8">
          Интерактивная платформа для создания и прохождения викторин
        </p>
        
        <div className="flex flex-col sm:flex-row gap-4 justify-center mt-8">
          <Link 
            href="/quizzes" 
            className="bg-blue-600 hover:bg-blue-700 text-white font-bold py-3 px-6 rounded-lg transition-colors"
          >
            Просмотреть викторины
          </Link>
          {!isAuthenticated && (
            <Link 
              href="/login" 
              className="bg-gray-200 hover:bg-gray-300 text-gray-800 font-bold py-3 px-6 rounded-lg transition-colors"
            >
              Войти в систему
            </Link>
          )}
        </div>
      </div>
      
      <div className="w-full max-w-3xl bg-white p-6 rounded-lg shadow-md">
        {isAuthenticated ? (
          <div>
            <h2 className="text-2xl font-bold mb-4">Ближайшая викторина</h2>
            
            {loading && (
              <div className="text-center py-4">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto"></div>
                <p className="mt-2">Загрузка викторин...</p>
              </div>
            )}
            
            {error && (
              <div className="bg-red-100 text-red-700 p-4 rounded-lg mb-4">
                {error}
              </div>
            )}
            
            {!loading && !error && upcomingQuiz ? (
              <div className="border rounded-lg p-4">
                <h3 className="text-xl font-bold">{upcomingQuiz.title}</h3>
                <p className="text-gray-600 mt-2">{upcomingQuiz.description}</p>
                <p className="text-blue-600 font-medium mt-2">
                  Запланировано на: {formatDate(upcomingQuiz.scheduled_time, DateFormat.MEDIUM)}
                </p>
                <div className="mt-4">
                  <Link 
                    href={`/quizzes/${upcomingQuiz.id}`} 
                    className="inline-block bg-blue-600 hover:bg-blue-700 text-white py-2 px-4 rounded transition-colors"
                  >
                    Подробнее
                  </Link>
                </div>
              </div>
            ) : (
              !loading && !error && (
                <p className="text-center py-4">Нет запланированных викторин.</p>
              )
            )}
          </div>
        ) : (
          <div className="text-center py-6">
            <p className="text-lg mb-4">Войдите в систему, чтобы увидеть предстоящие викторины.</p>
            <Link 
              href="/login" 
              className="inline-block bg-blue-600 hover:bg-blue-700 text-white py-2 px-4 rounded-lg transition-colors"
            >
              Войти
            </Link>
          </div>
        )}
      </div>
    </div>
  );
}
