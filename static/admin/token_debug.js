// Функционал для отладки JWT токенов
// Позволяет анализировать токены и определять причины проблем с аутентификацией

// DOM элементы
let tokenDebugSection;
let tokenDebugForm;
let tokenDebugResult;
let tokenDebugBtn;

// Функция для переключения между секциями
// Это локальная копия функции из app.js для использования внутри отладчика токенов
function switchSection(section) {
    console.log("Переключение секции на:", section ? section.id : "неизвестно");
    
    // Получаем все секции
    const sections = document.querySelectorAll('.section');
    sections.forEach(s => {
        if (s) {
            s.classList.add('hidden');
        }
    });
    
    // Скрываем все подформы
    const forms = document.querySelectorAll('.form-container, #add-questions-form, #schedule-quiz-form, #restart-quiz-form');
    forms.forEach(form => {
        if (form) form.classList.add('hidden');
    });
    
    // Активная вкладка навигации
    document.querySelectorAll('.nav-btn').forEach(btn => btn.classList.remove('active'));
    
    // Показываем нужную секцию
    if (section) {
        section.classList.remove('hidden');
        console.log("Показана секция:", section.id);
    }
}

// Показываем уведомление
function showToast(message, type = 'success', duration = 3000) {
    // Удаляем существующие уведомления
    const existingToasts = document.querySelectorAll('.toast');
    existingToasts.forEach(toast => toast.remove());
    
    // Создаем новое уведомление
    const toast = document.createElement('div');
    toast.className = `toast toast-${type}`;
    toast.textContent = message;
    
    document.body.appendChild(toast);
    
    // Автоматически удаляем через указанное время
    setTimeout(() => {
        toast.classList.add('toast-hide');
        // Полностью удаляем после окончания анимации
        setTimeout(() => toast.remove(), 300);
    }, duration);
}

// Инициализация функционала отладки токенов
function initTokenDebugger() {
    // Создаем кнопку в навигации админки
    const navMenu = document.querySelector('#admin-panel nav ul');
    
    if (navMenu) {
        const debugLi = document.createElement('li');
        tokenDebugBtn = document.createElement('button');
        tokenDebugBtn.id = 'token-debug-btn';
        tokenDebugBtn.className = 'nav-btn';
        tokenDebugBtn.textContent = 'Отладка токенов';
        tokenDebugBtn.addEventListener('click', showTokenDebugger);
        debugLi.appendChild(tokenDebugBtn);
        navMenu.appendChild(debugLi);
        console.log('Кнопка отладки токенов добавлена в меню');
    } else {
        console.error('Не найдено меню навигации для добавления кнопки отладки токенов');
    }
    
    // Создаем секцию отладки токенов
    tokenDebugSection = document.createElement('div');
    tokenDebugSection.id = 'token-debug-section';
    tokenDebugSection.className = 'section hidden';
    
    // Заголовок и описание
    const header = document.createElement('h2');
    header.textContent = 'Отладка JWT токенов';
    tokenDebugSection.appendChild(header);
    
    const description = document.createElement('p');
    description.textContent = 'Вставьте JWT токен для анализа и диагностики проблем с аутентификацией.';
    tokenDebugSection.appendChild(description);
    
    // Создаем форму отладки
    tokenDebugForm = document.createElement('form');
    tokenDebugForm.id = 'token-debug-form';
    tokenDebugForm.innerHTML = `
        <div class="form-group">
            <label for="token-input">JWT токен:</label>
            <textarea id="token-input" name="token" rows="4" required 
                placeholder="Вставьте JWT токен для анализа..."></textarea>
        </div>
        <div class="form-actions">
            <button type="button" id="parse-token-btn" class="btn btn-secondary">Локальный анализ</button>
            <button type="submit" id="debug-token-btn" class="btn btn-primary">Глубокий анализ на сервере</button>
        </div>
    `;
    tokenDebugSection.appendChild(tokenDebugForm);
    
    // Создаем блок для результатов
    tokenDebugResult = document.createElement('div');
    tokenDebugResult.id = 'token-debug-result';
    tokenDebugResult.className = 'token-debug-result hidden';
    tokenDebugSection.appendChild(tokenDebugResult);
    
    // Добавляем обработчики событий
    tokenDebugForm.addEventListener('submit', handleTokenAnalysis);
    document.getElementById('parse-token-btn')?.addEventListener('click', handleLocalTokenParse);
    
    // Добавляем секцию в админ-панель
    const mainContainer = document.querySelector('.main-container');
    if (mainContainer) {
        mainContainer.appendChild(tokenDebugSection);
        console.log('Секция отладки токенов добавлена в DOM');
    } else {
        console.error('Не найден контейнер для добавления секции отладки токенов');
    }
}

// Показать отладчик токенов
function showTokenDebugger() {
    tokenDebugResult.innerHTML = '';
    tokenDebugResult.classList.add('hidden');
    
    // Используем глобальную функцию из app.js, если доступна, иначе локальную
    if (window.appFunctions && window.appFunctions.switchSection) {
        window.appFunctions.switchSection(tokenDebugSection);
    } else {
        switchSection(tokenDebugSection);
    }
    
    // Активируем кнопку в меню
    document.querySelectorAll('.nav-btn').forEach(btn => btn.classList.remove('active'));
    tokenDebugBtn.classList.add('active');
}

// Локальный разбор токена (без запроса на сервер)
function handleLocalTokenParse() {
    const tokenInput = document.getElementById('token-input');
    const token = tokenInput.value.trim();
    
    if (!token) {
        // Используем глобальную функцию из app.js, если доступна, иначе локальную
        if (window.appFunctions && window.appFunctions.showToast) {
            window.appFunctions.showToast('Пожалуйста, введите токен', 'error');
        } else {
            showToast('Пожалуйста, введите токен', 'error');
        }
        return;
    }
    
    try {
        // Разбор JWT токена (без проверки подписи)
        const tokenParts = token.split('.');
        if (tokenParts.length !== 3) {
            throw new Error('Неверный формат JWT токена');
        }
        
        // Декодируем заголовок и полезную нагрузку токена
        const header = JSON.parse(atob(tokenParts[0]));
        const payload = JSON.parse(atob(tokenParts[1]));
        
        // Проверяем срок действия
        const isExpired = payload.exp && payload.exp < Math.floor(Date.now() / 1000);
        
        // Отображаем разобранные данные
        tokenDebugResult.innerHTML = `
            <h3>Результат локального анализа</h3>
            <div class="token-section">
                <h4>Заголовок (Header)</h4>
                <pre>${JSON.stringify(header, null, 2)}</pre>
            </div>
            <div class="token-section">
                <h4>Полезная нагрузка (Payload)</h4>
                <pre>${JSON.stringify(payload, null, 2)}</pre>
            </div>
            <div class="token-section">
                <h4>Метаданные</h4>
                <ul>
                    <li><strong>ID пользователя:</strong> ${payload.user_id || 'Не указан'}</li>
                    <li><strong>Email:</strong> ${payload.email || 'Не указан'}</li>
                    <li><strong>Время выдачи:</strong> ${payload.iat ? new Date(payload.iat * 1000).toLocaleString() : 'Не указано'}</li>
                    <li><strong>Время истечения:</strong> ${payload.exp ? new Date(payload.exp * 1000).toLocaleString() : 'Не указано'}</li>
                    <li><strong>Статус:</strong> ${isExpired ? '<span class="status-expired">Истек срок действия</span>' : '<span class="status-valid">Действителен</span>'}</li>
                </ul>
                <p class="note">Примечание: Локальный анализ не проверяет подпись и не может определить, был ли токен инвалидирован на сервере.</p>
            </div>
        `;
        tokenDebugResult.classList.remove('hidden');
    } catch (error) {
        console.error('Ошибка при разборе токена:', error);
        // Используем глобальную функцию из app.js, если доступна, иначе локальную
        if (window.appFunctions && window.appFunctions.showToast) {
            window.appFunctions.showToast('Ошибка при разборе токена: ' + error.message, 'error');
        } else {
            showToast('Ошибка при разборе токена: ' + error.message, 'error');
        }
    }
}

// Анализ токена с помощью сервера
async function handleTokenAnalysis(event) {
    event.preventDefault();
    
    const tokenInput = document.getElementById('token-input');
    const token = tokenInput.value.trim();
    
    if (!token) {
        // Используем глобальную функцию из app.js, если доступна, иначе локальную
        if (window.appFunctions && window.appFunctions.showToast) {
            window.appFunctions.showToast('Пожалуйста, введите токен', 'error');
        } else {
            showToast('Пожалуйста, введите токен', 'error');
        }
        return;
    }
    
    try {
        // Используем authService для выполнения авторизованного запроса
        // Это обеспечит автоматическое обновление токена при необходимости
        let response;
        try {
            // Пытаемся выполнить запрос через authService, если он доступен
            if (typeof authService !== 'undefined') {
                response = await authService._authorizedRequest('/api/auth/admin/debug-token', {
                    method: 'POST',
                    body: JSON.stringify({ token })
                });
            } else {
                // Fallback, если authService недоступен
                const res = await fetch('/api/auth/admin/debug-token', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': `Bearer ${localStorage.getItem('access_token')}`,
                        'X-CSRF-Token': localStorage.getItem('csrf_token')
                    },
                    body: JSON.stringify({ token }),
                    credentials: 'include'
                });
                
                if (!res.ok) {
                    throw new Error('Ошибка сервера при анализе токена');
                }
                
                response = await res.json();
            }
        } catch (authError) {
            console.error('Ошибка авторизации при анализе токена:', authError);
            // Используем глобальную функцию из app.js, если доступна, иначе локальную
            if (window.appFunctions && window.appFunctions.showToast) {
                window.appFunctions.showToast('Необходима авторизация для анализа токена', 'error');
            } else {
                showToast('Необходима авторизация для анализа токена', 'error');
            }
            
            // Перенаправляем на страницу входа через 2 секунды
            setTimeout(() => {
                window.location.href = '/admin/login.html?redirect=' + encodeURIComponent(window.location.pathname);
            }, 2000);
            return;
        }
        
        // Форматируем и выводим результат
        formatTokenDebugResult(response);
    } catch (error) {
        console.error('Ошибка при анализе токена:', error);
        // Используем глобальную функцию из app.js, если доступна, иначе локальную
        if (window.appFunctions && window.appFunctions.showToast) {
            window.appFunctions.showToast('Ошибка при анализе токена: ' + (error.message || 'Ошибка сервера'), 'error');
        } else {
            showToast('Ошибка при анализе токена: ' + (error.message || 'Ошибка сервера'), 'error');
        }
    }
}

// Форматирование результата отладки токена
function formatTokenDebugResult(result) {
    // Определяем статус токена
    let status = 'unknown';
    let statusText = 'Неизвестно';
    
    if (result.is_invalidated) {
        status = 'invalidated';
        statusText = 'Инвалидирован';
    } else if (result.expired) {
        status = 'expired';
        statusText = 'Истек срок действия';
    } else if (result.valid === false && !result.expired && !result.is_invalidated) {
        status = 'invalid';
        statusText = 'Недействителен (неверная подпись)';
    } else if (result.valid === false) {
        status = 'invalid';
        statusText = 'Недействителен';
    }
    
    // Создаем HTML для отображения результата
    tokenDebugResult.innerHTML = `
        <h3>Результат анализа токена на сервере</h3>
        <div class="token-status">
            <span class="token-status-label">Статус токена:</span>
            <span class="token-status-value status-${status}">${statusText}</span>
        </div>
        
        <div class="token-section">
            <h4>Информация о пользователе</h4>
            <ul>
                <li><strong>ID пользователя:</strong> ${result.user_id || 'Не указан'}</li>
                <li><strong>Email:</strong> ${result.email || 'Не указан'}</li>
            </ul>
        </div>
        
        <div class="token-section">
            <h4>Временные метки</h4>
            <ul>
                <li><strong>Время выдачи:</strong> ${result.issued_at ? new Date(result.issued_at).toLocaleString() : 'Не указано'}</li>
                <li><strong>Время истечения:</strong> ${result.expires_at ? new Date(result.expires_at).toLocaleString() : 'Не указано'}</li>
                ${result.invalidation_time ? `<li><strong>Время инвалидации:</strong> ${new Date(result.invalidation_time).toLocaleString()}</li>` : ''}
            </ul>
        </div>
        
        <div class="token-section">
            <h4>Полные данные</h4>
            <pre>${JSON.stringify(result, null, 2)}</pre>
        </div>
        
        ${result.is_invalidated ? `
        <div class="token-section">
            <h4>Почему токен инвалидирован?</h4>
            <p>Этот токен был инвалидирован на сервере. Возможные причины:</p>
            <ul>
                <li>Выход пользователя из системы (logout)</li>
                <li>Сброс пароля пользователя</li>
                <li>Выход пользователя из всех устройств</li>
                <li>Административное действие (блокировка токена)</li>
            </ul>
            <p>Время инвалидации: ${new Date(result.invalidation_time).toLocaleString()}</p>
        </div>
        ` : ''}
    `;
    
    tokenDebugResult.classList.remove('hidden');
}

// Инициализируем отладчик токенов после загрузки страницы
document.addEventListener('DOMContentLoaded', function() {
    // Инициализируем отладчик токенов только после полной загрузки DOM
    console.log('Инициализация отладчика токенов');
    initTokenDebugger();
}); 