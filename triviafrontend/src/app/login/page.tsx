"use client";

import Link from "next/link";
import { useState } from "react";
import { useAuth } from "../../lib/auth/auth-context";
import { useRouter } from "next/navigation";

export default function LoginPage() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [formError, setFormError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  
  const { login, error, clearError } = useAuth();
  const router = useRouter();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!email || !password) {
      setFormError("Пожалуйста, заполните все поля");
      return;
    }
    
    try {
      setIsSubmitting(true);
      setFormError(null);
      clearError();
      
      await login(email, password);
      
      // Успешный вход - перенаправляем на главную
      router.push("/");
    } catch (err) {
      console.error("Ошибка входа:", err);
      // Ошибка уже будет установлена в контексте аутентификации
      setFormError(error || "Произошла ошибка при входе в систему");
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="max-w-md mx-auto mt-10">
      <h1 className="text-3xl font-bold text-center mb-6">Вход в систему</h1>
      <div className="bg-white p-8 rounded-lg shadow-md">
        {(formError || error) && (
          <div className="mb-6 p-3 rounded bg-red-50 text-red-600 text-sm">
            {formError || error}
          </div>
        )}
        
        <form onSubmit={handleSubmit} className="space-y-6">
          <div>
            <label htmlFor="email" className="block text-sm font-medium text-gray-700 mb-1">
              Email
            </label>
            <input
              id="email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full px-4 py-2 border rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="Введите ваш email"
              required
              disabled={isSubmitting}
            />
          </div>

          <div>
            <label htmlFor="password" className="block text-sm font-medium text-gray-700 mb-1">
              Пароль
            </label>
            <input
              id="password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full px-4 py-2 border rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="Введите пароль"
              required
              disabled={isSubmitting}
            />
          </div>

          <div>
            <button
              type="submit"
              className={`w-full bg-blue-600 hover:bg-blue-700 text-white font-bold py-2 px-4 rounded-md transition-colors ${
                isSubmitting ? "opacity-70 cursor-not-allowed" : ""
              }`}
              disabled={isSubmitting}
            >
              {isSubmitting ? "Выполняется вход..." : "Войти"}
            </button>
          </div>
        </form>

        <div className="mt-4 text-center text-sm text-gray-600">
          Нет аккаунта?{" "}
          <Link href="/register" className="text-blue-600 hover:underline">
            Зарегистрироваться
          </Link>
        </div>
      </div>
    </div>
  );
} 