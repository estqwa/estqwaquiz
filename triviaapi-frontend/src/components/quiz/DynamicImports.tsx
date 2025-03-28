import dynamic from 'next/dynamic';

/**
 * Динамический импорт компонента ResultsPanel с отложенной загрузкой
 * Это позволяет уменьшить размер начального бандла и ускорить загрузку страницы
 */
export const DynamicResultsPanel = dynamic(
  () => import('./ResultsPanel'),
  {
    loading: () => (
      <div className="bg-white rounded-lg shadow-sm p-6">
        <div className="flex justify-center items-center h-40">
          <div className="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-blue-500"></div>
        </div>
      </div>
    ),
    ssr: false // Отключаем SSR для этого компонента, так как он используется только после загрузки страницы
  }
);

/**
 * Динамический импорт компонента ActiveQuiz с отложенной загрузкой
 */
export const DynamicActiveQuiz = dynamic(
  () => import('./ActiveQuiz'),
  {
    loading: () => (
      <div className="bg-white rounded-lg shadow-sm p-6">
        <div className="text-center">
          <div className="animate-spin inline-block h-8 w-8 border-t-2 border-b-2 border-blue-500 rounded-full"></div>
          <p className="mt-2 text-gray-500">Загрузка викторины...</p>
        </div>
      </div>
    ),
    ssr: false
  }
);

/**
 * Динамический импорт компонента QuestionCard с отложенной загрузкой
 */
export const DynamicQuestionCard = dynamic(
  () => import('./QuestionCard'),
  {
    loading: () => (
      <div className="bg-white rounded-lg shadow-sm p-4">
        <div className="animate-pulse">
          <div className="h-4 bg-gray-200 rounded w-3/4 mb-6"></div>
          <div className="h-4 bg-gray-200 rounded w-full mb-4"></div>
          <div className="h-4 bg-gray-200 rounded w-5/6 mb-8"></div>
          
          <div className="space-y-3">
            <div className="h-10 bg-gray-200 rounded"></div>
            <div className="h-10 bg-gray-200 rounded"></div>
            <div className="h-10 bg-gray-200 rounded"></div>
            <div className="h-10 bg-gray-200 rounded"></div>
          </div>
          
          <div className="flex justify-end mt-6">
            <div className="h-10 bg-gray-200 rounded w-1/4"></div>
          </div>
        </div>
      </div>
    ),
    ssr: false
  }
); 