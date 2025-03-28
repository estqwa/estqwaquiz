# Trivia API Frontend

Фронтенд-приложение для Trivia API, разработанное на стеке React/Next.js/TypeScript.

## Технологии

- **React** - библиотека для создания пользовательского интерфейса
- **Next.js** - React-фреймворк для SSR/SSG, роутинга и оптимизаций
- **TypeScript** - для статической типизации
- **React Query** - для управления серверным состоянием
- **Redux Toolkit** - для управления глобальным состоянием
- **TailwindCSS** - для стилизации
- **Axios** - HTTP-клиент для взаимодействия с REST API
- **WebSocket API** - для коммуникации в реальном времени

## Установка и запуск

1. Клонировать репозиторий
2. Установить зависимости:
   ```bash
   npm install
   ```
3. Запустить в режиме разработки:
   ```bash
   npm run dev
   ```
4. Открыть [http://localhost:3000](http://localhost:3000) в браузере

## Структура проекта

- `src/api/` - API клиенты и сервисы
- `src/components/` - React компоненты
- `src/hooks/` - Кастомные React хуки
- `src/pages/` - Страницы приложения (роутинг Next.js)
- `src/store/` - Redux Toolkit хранилище
- `src/types/` - TypeScript типы и интерфейсы
- `src/utils/` - Вспомогательные функции
- `src/constants/` - Константы приложения 