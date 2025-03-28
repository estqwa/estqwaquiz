import React, { Component, ErrorInfo, ReactNode } from 'react';
import Button from './common/Button';

interface ErrorBoundaryProps {
  children: ReactNode;
  fallback?: ReactNode;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

/**
 * Компонент ErrorBoundary для отлавливания ошибок рендеринга в дочерних компонентах
 * и отображения запасного UI вместо сломанного дерева компонентов
 */
class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = {
      hasError: false,
      error: null
    };
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    // Обновляем состояние, чтобы следующий рендер показал запасной UI
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo): void {
    // Можно также отправить ошибку в сервис аналитики
    console.error('ErrorBoundary caught an error:', error, errorInfo);
  }

  resetErrorBoundary = (): void => {
    this.setState({
      hasError: false,
      error: null
    });
  };

  render(): ReactNode {
    if (this.state.hasError) {
      // Если передан пользовательский fallback, используем его
      if (this.props.fallback) {
        return this.props.fallback;
      }

      // Иначе используем стандартный fallback
      return (
        <div className="min-h-screen bg-gray-50 flex flex-col justify-center py-12 sm:px-6 lg:px-8">
          <div className="mt-8 sm:mx-auto sm:w-full sm:max-w-md">
            <div className="bg-white py-8 px-4 shadow sm:rounded-lg sm:px-10">
              <div className="text-center">
                <svg
                  className="mx-auto h-12 w-12 text-red-500"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                  aria-hidden="true"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth="2"
                    d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
                  />
                </svg>
                <h3 className="mt-4 text-lg font-medium text-gray-900">Произошла ошибка</h3>
                <p className="mt-2 text-sm text-gray-500">
                  К сожалению, в приложении произошла непредвиденная ошибка.
                </p>
                {this.state.error && (
                  <div className="mt-3 bg-red-50 p-4 rounded-md">
                    <p className="text-sm text-red-700 whitespace-pre-wrap break-words">
                      {this.state.error.message}
                    </p>
                  </div>
                )}
                <div className="mt-5">
                  <Button
                    type="button"
                    onClick={this.resetErrorBoundary}
                    variant="primary"
                    className="w-full"
                  >
                    Попробовать снова
                  </Button>
                  <Button
                    type="button"
                    onClick={() => window.location.href = '/'}
                    variant="outline"
                    className="w-full mt-3"
                  >
                    Вернуться на главную
                  </Button>
                </div>
              </div>
            </div>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}

export default ErrorBoundary; 