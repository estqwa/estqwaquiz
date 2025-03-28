# Резюме проекта Trivia Quiz Frontend

## Общая структура проекта

Мы создали фронтенд-приложение для системы проведения онлайн-викторин в реальном времени, используя методологию Feature-Sliced Design (FSD). Приложение построено на React и TypeScript с использованием Vite в качестве инструмента сборки.

## Технологический стек

- **Фреймворк**: React с TypeScript
- **Сборка**: Vite
- **Маршрутизация**: React Router
- **Управление состоянием**: Redux Toolkit
- **HTTP-клиент**: Axios
- **WebSocket**: Нативный WebSocket API
- **Стилизация**: Tailwind CSS
- **Формы**: React Hook Form + Zod
- **Уведомления**: Sonner
- **Работа с датами**: date-fns
- **Вспомогательные библиотеки**: clsx, tailwind-merge, class-variance-authority

## Структура файлов

### Корневые файлы
- `index.html` - Главный HTML-файл
- `.env` - Переменные окружения
- `package.json` - Зависимости проекта
- `tailwind.config.js` - Конфигурация Tailwind CSS
- `postcss.config.js` - Конфигурация PostCSS

### Исходный код (src)

- `src/main.tsx` - Точка входа приложения
- `src/index.css` - Глобальные стили

### Слой приложения (`app`)
- `src/app/App.tsx` - Корневой компонент приложения
- `src/app/providers/query-provider.tsx` - Провайдер для TanStack Query
- `src/app/routes/index.tsx` - Конфигурация маршрутов
- `src/app/routes/protected-route.tsx` - Компонент для защищенных маршрутов
- `src/app/store/index.ts` - Настройка Redux store

### Слой страниц (`pages`)
- `src/pages/home/index.tsx` - Главная страница
- `src/pages/auth/login/index.tsx` - Страница входа
- `src/pages/auth/register/index.tsx` - Страница регистрации
- `src/pages/quiz/waiting-room/index.tsx` - Страница зала ожидания викторины
- `src/pages/quiz/active/index.tsx` - Страница активной викторины
- `src/pages/quiz/results/index.tsx` - Страница результатов викторины
- `src/pages/profile/index.tsx` - Страница профиля пользователя

### Слой виджетов (`widgets`)
- `src/widgets/header/index.tsx` - Компонент шапки сайта
- `src/widgets/layouts/main-layout.tsx` - Основной лейаут приложения
- `src/widgets/timer/quiz-timer.tsx` - Компонент таймера для вопросов
- `src/widgets/quiz-progress/quiz-progress.tsx` - Компонент прогресса викторины
- `src/widgets/leaderboard/leaderboard.tsx` - Компонент таблицы лидеров

### Слой фич (`features`)
- `src/features/auth/login-form/login-form.tsx` - Форма входа
- `src/features/auth/register-form/register-form.tsx` - Форма регистрации
- `src/features/quiz/quiz-list/quiz-list.tsx` - Список доступных викторин
- `src/features/quiz/waiting-room/waiting-room.tsx` - Компонент зала ожидания
- `src/features/quiz/question-display/question-display.tsx` - Отображение вопроса
- `src/features/quiz/quiz-websocket-listener/quiz-websocket-listener.tsx` - Обработчик WebSocket событий

### Слой сущностей (`entities`)
- `src/entities/user/model/types.ts` - Типы пользователя
- `src/entities/user/model/auth-slice.ts` - Redux-слайс для авторизации
- `src/entities/user/model/use-auth.ts` - Хук для работы с авторизацией
- `src/entities/quiz/model/types.ts` - Типы викторины
- `src/entities/quiz/model/quiz-slice.ts` - Redux-слайс для викторины
- `src/entities/quiz/api/quiz-service.ts` - Сервис API для работы с викторинами
- `src/entities/quiz/ui/quiz-card.tsx` - Компонент карточки викторины

### Слой общих ресурсов (`shared`)
- `src/shared/api/rest/api-client.ts` - Настройка Axios-клиента
- `src/shared/api/websocket/use-websocket.ts` - Хук для работы с WebSocket
- `src/shared/api/websocket/use-trivia-socket.ts` - Хук для работы с WebSocket викторины
- `src/shared/api/websocket/events.ts` - Типы событий WebSocket
- `src/shared/ui/button/button.tsx` - Компонент кнопки
- `src/shared/ui/input/input.tsx` - Компонент поля ввода
- `src/shared/lib/utils.ts` - Вспомогательные функции

## Маршруты приложения

| Путь | Компонент | Описание |
|------|-----------|----------|
| `/` | `HomePage` | Главная страница со списком викторин |
| `/login` | `LoginPage` | Страница входа |
| `/register` | `RegisterPage` | Страница регистрации |
| `/quiz/waiting-room/:id` | `QuizWaitingRoomPage` | Зал ожидания перед началом викторины |
| `/quiz/active/:id` | `QuizActivePage` | Активная викторина |
| `/quiz/results/:id` | `QuizResultsPage` | Результаты викторины |
| `/profile` | `ProfilePage` | Профиль пользователя |

## API-эндпоинты

### Аутентификация
- `POST /api/auth/register` - Регистрация нового пользователя
- `POST /api/auth/login` - Вход в систему

### Пользователи
- `GET /api/users/me` - Получение данных текущего пользователя
- `PUT /api/users/me` - Обновление данных пользователя

### Викторины
- `GET /api/quizzes` - Получение списка викторин
- `GET /api/quizzes/active` - Получение активной викторины
- `GET /api/quizzes/scheduled` - Получение запланированных викторин
- `GET /api/quizzes/:id` - Получение информации о конкретной викторине
- `GET /api/quizzes/:id/results` - Получение результатов викторины
- `GET /api/quizzes/:id/my-result` - Получение результата текущего пользователя

## WebSocket-события

### События от клиента к серверу
- `user:ready` - Пользователь готов к викторине
- `user:answer` - Ответ пользователя на вопрос
- `user:heartbeat` - Проверка соединения

### События от сервера к клиенту
- `quiz:announcement` - Анонс викторины
- `quiz:waiting_room` - Открытие зала ожидания
- `quiz:countdown` - Обратный отсчет
- `quiz:start` - Начало викторины
- `quiz:question` - Новый вопрос
- `quiz:timer` - Обновление таймера
- `quiz:answer_reveal` - Показ правильного ответа
- `quiz:answer_result` - Результат ответа
- `quiz:end` - Конец викторины
- `quiz:leaderboard` - Таблица лидеров
- `quiz:user_ready` - Уведомление о готовности пользователя
- `quiz:cancelled` - Уведомление об отмене викторины
- `server:heartbeat` - Ответ на проверку соединения
- `error` - Сообщение об ошибке

## Ключевые компоненты состояния

### Состояние аутентификации (auth)
- `user` - Данные пользователя
- `token` - JWT токен
- `isAuthenticated` - Флаг авторизации

### Состояние викторины (quiz)
- `activeQuiz` - Активная викторина
- `currentQuestion` - Текущий вопрос
- `remainingSeconds` - Оставшееся время
- `userAnswers` - Ответы пользователя
- `correctAnswers` - Правильные ответы
- `leaderboard` - Таблица лидеров

## Примечания по разработке

При запуске проекта необходимо обратить внимание на:

1. **CORS**: Необходимо настроить CORS на сервере для обработки OPTIONS-запросов
2. **WebSocket**: WebSocket-соединение должно быть защищено токеном JWT
3. **Зависимости**: Убедитесь, что все зависимости установлены: class-variance-authority, clsx, tailwind-merge и др.
4. **Структура директорий**: Важно соблюдать структуру директорий согласно FSD

## Запуск проекта

```bash
# Установка зависимостей
npm install
# или
pnpm install

# Запуск режима разработки
npm run dev
# или
pnpm dev
```

По умолчанию, приложение доступно по адресу http://localhost:5173
