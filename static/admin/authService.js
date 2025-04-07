/**
 * Сервис аутентификации для управления токенами и сессиями
 */
class AuthService {
    constructor() {
        this.accessToken = localStorage.getItem('access_token') || null;
        this.csrfToken = localStorage.getItem('csrf_token') || null;
        this.tokenExpiry = localStorage.getItem('token_expiry') 
            ? new Date(localStorage.getItem('token_expiry')) 
            : null;
        this.refreshInProgress = false;
        this.refreshQueue = [];
        
        // Исправляем ошибку парсинга JSON
        try {
            const userData = localStorage.getItem('user');
            this.user = userData ? JSON.parse(userData) : null;
        } catch (e) {
            console.error('Ошибка при парсинге данных пользователя:', e);
            this.user = null;
            // Очищаем некорректные данные
            localStorage.removeItem('user');
        }
        
        this.currentSessionId = localStorage.getItem('session_id') || null;

        // Признак первой инициализации
        this.initialized = false;

        // Определяем режим аутентификации
        // useCookieAuth = true, если есть csrfToken и нет accessToken, или если в localStorage есть флаг
        this.useCookieAuth = localStorage.getItem('use_cookie_auth') === 'true' || 
                            (!!(this.csrfToken) && !this.accessToken);
        // Сохраняем обновленное значение в localStorage
        localStorage.setItem('use_cookie_auth', this.useCookieAuth.toString());
        console.log('Режим аутентификации при инициализации:', 
                  this.useCookieAuth ? 'Cookie Auth' : 'Bearer Auth', 
                  { hasCsrf: !!this.csrfToken, hasAccess: !!this.accessToken, storedInLocalStorage: true });

        // Базовый URL API
        this.apiBaseUrl = '/api';

        // Таймер для проверки истечения токена
        this.tokenCheckInterval = null;
        
        // WebSocket соединение
        this.ws = null;
        this.wsReconnectAttempts = 0;
        this.wsMaxReconnectAttempts = 5;
        this.wsReconnectInterval = 5000; // 5 секунд между попытками
        this.wsDisconnected = false;  // Флаг, показывающий, что соединение разорвано
        
        // Лимит сессий
        this.sessionLimit = localStorage.getItem('session_limit') 
            ? parseInt(localStorage.getItem('session_limit')) 
            : 10;
        this.sessionCount = 0;
        
        // Инициализация обработки событий
        this._setupEventListeners();
    }

    /**
     * Инициализирует сервис аутентификации и проверяет статус сессии
     */
    async init() {
        if (this.initialized) {
            return;
        }

        console.log('Инициализация AuthService...');
        console.log('Текущий CSRF токен:', this.csrfToken);
        console.log('Текущий токен доступа:', this.accessToken ? 'установлен' : 'отсутствует');

        // Проверяем, действителен ли текущий токен
        if ((this.accessToken || this.csrfToken) && this.tokenExpiry) {
            // Если токен истек, пробуем обновить
            if (new Date() >= this.tokenExpiry) {
                console.log('Токен истек, пробуем обновить...');
                try {
                    await this.refreshToken();
                } catch (error) {
                    console.error('Ошибка при инициализации токена:', error);
                    this._clearAuthData();
                    return false;
                }
            } else {
                console.log('Токен действителен, срок действия до:', this.tokenExpiry);
                // Проверяем наличие CSRF токена для Cookie Auth
                if (this.useCookieAuth && !this.csrfToken) {
                    console.warn('CSRF токен отсутствует, пробуем обновить токен...');
                    try {
                        await this.refreshToken();
                    } catch (error) {
                        console.error('Ошибка при получении CSRF токена:', error);
                        this._clearAuthData();
                        return false;
                    }
                }
                
                // Начинаем проверку срока действия токена
                this._startTokenCheck();
            }
            
            // Подключаемся к WebSocket для получения уведомлений
            try {
                await this._connectWebSocket();
            } catch (wsError) {
                console.error('Ошибка при подключении к WebSocket:', wsError);
                // Продолжаем работу, даже если WebSocket не подключился
            }
        } else {
            console.log('Токен доступа отсутствует или истек');
            this._clearAuthData();
            return false;
        }

        this.initialized = true;
        return true;
    }

    /**
     * Проверяет, аутентифицирован ли пользователь
     * @returns {boolean} Статус аутентификации
     */
    isAuthenticated() {
        // В режиме Cookie Auth мы проверяем наличие CSRF токена вместо access токена
        if (this.useCookieAuth) {
            return !!this.csrfToken && !!this.user && new Date() < this.tokenExpiry;
        }
        // В режиме Bearer Auth проверяем access токен
        return !!this.accessToken && !!this.user && new Date() < this.tokenExpiry;
    }

    /**
     * Проверяет, является ли пользователь администратором
     * @returns {boolean} Статус администратора
     */
    isAdmin() {
        return this.isAuthenticated() && this.user && this.user.is_admin === true;
    }

    /**
     * Выполняет вход пользователя
     * @param {string} email Email пользователя
     * @param {string} password Пароль пользователя
     * @returns {Promise<Object>} Данные пользователя
     */
    async login(email, password) {
        try {
            const response = await fetch(`${this.apiBaseUrl}/auth/login`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ email, password }),
                credentials: 'include' // Важно для получения куки
            });

            if (!response.ok) {
                const errorData = await response.json();
                throw new Error(errorData.error || 'Ошибка входа');
            }

            const data = await response.json();
            this._setAuthData(data);
            this._startTokenCheck();
            
            // Сохраняем режим аутентификации в localStorage
            localStorage.setItem('use_cookie_auth', this.useCookieAuth.toString());
            
            // Выводим отладочную информацию
            console.log('Успешная авторизация! Токены обновлены.');
            console.log('CSRF токен:', this.csrfToken);
            console.log('Токен доступа установлен, срок действия до:', this.tokenExpiry);
            console.log('Режим аутентификации после входа:', this.useCookieAuth ? 'Cookie Auth' : 'Bearer Auth');
            
            // Подключаемся к WebSocket после входа
            try {
                await this._connectWebSocket();
            } catch (wsError) {
                console.error('Ошибка при подключении к WebSocket после входа:', wsError);
                // Продолжаем работу, даже если WebSocket не подключился
            }
            
            // Перенаправляем пользователя на главную страницу админки (без параметра showLogin)
            window.location.href = 'index.html';

            return this.user;
        } catch (error) {
            console.error('Ошибка входа:', error);
            throw error;
        }
    }

    /**
     * Выполняет выход пользователя
     * @returns {Promise<void>}
     */
    async logout() {
        try {
            if (this.accessToken) {
                await fetch(`${this.apiBaseUrl}/auth/logout`, {
                    method: 'POST',
                    headers: this._getAuthHeaders(),
                    credentials: 'include'
                });
            }
        } catch (error) {
            console.error('Ошибка при выходе:', error);
        } finally {
            this._clearAuthData();
            this._disconnectWebSocket();
            window.location.href = 'index.html?showLogin=true';
        }
    }

    /**
     * Выполняет выход со всех устройств
     * @returns {Promise<void>}
     */
    async logoutAllDevices() {
        try {
            if (this.accessToken) {
                await fetch(`${this.apiBaseUrl}/auth/logout-all`, {
                    method: 'POST',
                    headers: this._getAuthHeaders(),
                    credentials: 'include'
                });
            }
        } catch (error) {
            console.error('Ошибка при выходе со всех устройств:', error);
        } finally {
            this._clearAuthData();
            this._disconnectWebSocket();
            window.location.href = 'index.html?showLogin=true';
        }
    }

    /**
     * Обновляет токен доступа
     * @returns {Promise<void>}
     */
    async refreshToken() {
        // Если обновление уже выполняется, добавляем в очередь
        if (this.refreshInProgress) {
            return new Promise((resolve, reject) => {
                this.refreshQueue.push({ resolve, reject });
            });
        }

        this.refreshInProgress = true;

        try {
            const response = await fetch(`${this.apiBaseUrl}/auth/refresh`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-CSRF-Token': this.csrfToken || ''
                },
                credentials: 'include'
            });

            if (!response.ok) {
                const errorData = await response.json();
                console.error('Ошибка обновления токена:', errorData);
                throw new Error(`Ошибка обновления токена: ${errorData.error_type || 'unknown error'}`);
            }

            const data = await response.json();
            this._setAuthData(data);

            // Обрабатываем очередь запросов
            this.refreshQueue.forEach(request => request.resolve());
            console.log('Токен успешно обновлен. Новое время истечения:', this.tokenExpiry);
        } catch (error) {
            console.error('Ошибка при обновлении токена:', error);
            // Отклоняем все запросы в очереди
            this.refreshQueue.forEach(request => request.reject(error));
            this._clearAuthData();
            window.location.href = 'index.html?showLogin=true&error=session_expired';
        } finally {
            this.refreshInProgress = false;
            this.refreshQueue = [];
        }
    }

    /**
     * Получает активные сессии пользователя
     * @returns {Promise<Array>} Список активных сессий
     */
    async getActiveSessions() {
        const response = await this._authorizedRequest(`${this.apiBaseUrl}/auth/sessions`);
        
        // Сохраняем количество сессий
        if (response && response.sessions) {
            this.sessionCount = response.sessions.length;
            
            // Проверяем, не превышен ли лимит сессий
            if (this.sessionCount >= this.sessionLimit) {
                this._showSessionLimitWarning();
            }
        }
        
        return response;
    }

    /**
     * Получает WebSocket тикет для подключения в режиме Cookie Auth
     * @param {number} retryCount Количество повторных попыток
     * @param {boolean} tryAlternativePath Попытаться использовать альтернативный путь
     * @returns {Promise<string>} WebSocket тикет
     */
    async getWsTicket(retryCount = 0, tryAlternativePath = false) {
        try {
            console.log('Отправка запроса на получение WebSocket тикета...');
            
            // Для отладки добавляем параметр, чтобы обойти кэширование
            const timestamp = new Date().getTime();
            
            // Выбираем путь в зависимости от параметра tryAlternativePath
            let url;
            // tryAlternativePath может быть число (индекс пути) или boolean
            if (typeof tryAlternativePath === 'number') {
                // Массив всех возможных путей для endpoint ws-ticket
                const paths = [
                    `${this.apiBaseUrl}/auth/ws-ticket`, // Исходный путь
                    `${this.apiBaseUrl}/auth/wsticket`,  // Альтернативный путь 1
                    `${this.apiBaseUrl}/ws-ticket`,      // Путь без auth/ 
                    `/ws-ticket`                         // Прямой путь
                ];
                
                const pathIndex = tryAlternativePath;
                if (pathIndex < paths.length) {
                    url = `${paths[pathIndex]}?_=${timestamp}`;
                    console.log(`Пробуем путь для WS-тикета #${pathIndex + 1}:`, url);
                } else {
                    throw new Error('Все возможные пути для WS-тикета испробованы');
                }
            } else if (tryAlternativePath === true) {
                // Для обратной совместимости - используем альтернативный путь из предыдущей версии
                url = `${this.apiBaseUrl}/auth/wsticket?_=${timestamp}`;
                console.log('Используем альтернативный путь для WS-тикета:', url);
            } else {
                // Исходный путь
                url = `${this.apiBaseUrl}/auth/ws-ticket?_=${timestamp}`;
                console.log('Используем основной путь для WS-тикета:', url);
            }
            
            try {
                // Используем fetch напрямую, чтобы лучше обрабатывать ошибки
                const response = await fetch(url, {
                    method: 'POST',  // Используем POST, как в маршруте
                    headers: this._getAuthHeaders(),
                    credentials: 'include'
                });
                
                if (!response.ok) {
                    // Если это 404, и мы ещё не перепробовали все пути
                    if (response.status === 404) {
                        // Если это был первый вызов или явно указан boolean
                        if (tryAlternativePath === false) {
                            console.warn('Эндпоинт не найден (404), пробуем по порядку все альтернативные пути');
                            return this.getWsTicket(retryCount, 1); // Начинаем с индекса 1 (второй путь)
                        } else if (typeof tryAlternativePath === 'number') {
                            // Пробуем следующий путь
                            console.warn(`Путь #${tryAlternativePath + 1} не найден, пробуем следующий...`);
                            return this.getWsTicket(retryCount, tryAlternativePath + 1);
                        }
                    }
                    
                    const errorText = await response.text();
                    console.error(`Ошибка при получении WS-тикета: ${response.status} ${response.statusText}`, errorText);
                    throw new Error(`HTTP ошибка: ${response.status}`);
                }
                
                // Пробуем разобрать JSON ответ
                const data = await response.json();
                console.log('Ответ от сервера на запрос WS-тикета:', data);
                
                // Проверяем различные форматы ответа от сервера
                // Формат 1: { ticket: "значение" }
                // Формат 2: { data: { ticket: "значение" }, success: true }
                let ticket = null;
                
                if (data && data.ticket) {
                    // Формат 1: напрямую в корне
                    ticket = data.ticket;
                    console.log('WebSocket тикет найден в корне ответа');
                } else if (data && data.data && data.data.ticket) {
                    // Формат 2: внутри свойства data
                    ticket = data.data.ticket;
                    console.log('WebSocket тикет найден внутри свойства data');
                } else if (data && data.success === false) {
                    console.error('Сервер вернул ошибку:', data.error || 'Неизвестная ошибка');
                    throw new Error(data.error || 'Ошибка получения WebSocket тикета');
                } else {
                    console.error('Неожиданный формат ответа:', data);
                    throw new Error('Не удалось получить WebSocket тикет (неверный формат ответа)');
                }
                
                if (ticket) {
                    console.log('WebSocket тикет успешно получен');
                    return ticket;
                } else {
                    throw new Error('WebSocket тикет отсутствует в ответе сервера');
                }
            } catch (fetchError) {
                console.error('Ошибка при запросе WS-тикета:', fetchError);
                
                // Если это 404 и мы еще не перепробовали все пути
                if (fetchError.message.includes('404')) {
                    if (tryAlternativePath === false) {
                        console.warn('Ошибка 404, пробуем альтернативные пути');
                        return this.getWsTicket(retryCount, 1); // Начинаем с индекса 1
                    } else if (typeof tryAlternativePath === 'number' && tryAlternativePath < 3) {
                        // Пробуем следующий путь
                        console.warn(`Путь #${tryAlternativePath + 1} не найден, пробуем следующий...`);
                        return this.getWsTicket(retryCount, tryAlternativePath + 1);
                    }
                }
                
                throw fetchError;
            }
        } catch (error) {
            console.error('Ошибка при получении WebSocket тикета:', error);
            
            // Если это не последняя попытка, пробуем еще раз
            if (retryCount < 2) {
                console.log(`Повторная попытка получения WebSocket тикета (${retryCount + 1}/3)...`);
                // Добавляем небольшую задержку перед повторной попыткой
                await new Promise(resolve => setTimeout(resolve, 1000));
                return this.getWsTicket(retryCount + 1, tryAlternativePath);
            }
            
            // В случае неустранимой ошибки, возвращаем fallback-токен (access_token)
            // Это позволит работать WebSocket в режиме Bearer даже при отсутствии эндпоинта тикета
            if (this.accessToken) {
                console.warn('Не удалось получить WS-тикет, используем fallback на access_token для WebSocket');
                return this.accessToken;
            }
            
            throw error;
        }
    }

    /**
     * Получает информацию о лимите сессий
     * @returns {Promise<Object>} Информация о лимите сессий
     */
    async getSessionLimit() {
        try {
            const response = await this._authorizedRequest(`${this.apiBaseUrl}/auth/session-limit`);
            if (response && response.limit) {
                this.sessionLimit = response.limit;
                localStorage.setItem('session_limit', this.sessionLimit.toString());
            }
            return response;
        } catch (error) {
            console.error('Ошибка при получении лимита сессий:', error);
            return { limit: this.sessionLimit };
        }
    }

    /**
     * Отзывает конкретную сессию
     * @param {number} sessionId ID сессии для отзыва
     * @returns {Promise<Object>} Результат операции
     */
    async revokeSession(sessionId, reason = 'user_revoked') {
        return this._authorizedRequest(`${this.apiBaseUrl}/auth/revoke-session?reason=${encodeURIComponent(reason)}`, {
            method: 'POST',
            body: JSON.stringify({ session_id: sessionId })
        });
    }

    /**
     * Выполняет аутентифицированный запрос к API
     * @param {string} url URL запроса
     * @param {Object} options Опции запроса
     * @returns {Promise<any>} Результат запроса
     */
    async _authorizedRequest(url, options = {}) {
        // Проверяем, нужно ли обновить токен
        if (this.tokenExpiry && new Date() > new Date(this.tokenExpiry.getTime() - 5 * 60 * 1000)) {
            console.log('Токен скоро истечет, выполняем автоматическое обновление...');
            await this.refreshToken();
        }

        const requestOptions = {
            ...options,
            headers: {
                ...this._getAuthHeaders(),
                ...(options.headers || {})
            },
            credentials: 'include'
        };

        try {
            const response = await fetch(url, requestOptions);
            
            // Если токен истек, обновляем и повторяем запрос
            if (response.status === 401) {
                const errorData = await response.json();
                console.log('Получен 401 ответ:', errorData);
                
                if (errorData.error_type === 'token_expired') {
                    console.log('Токен истек, выполняем автоматическое обновление...');
                    await this.refreshToken();
                    // Обновляем заголовки с новым токеном
                    requestOptions.headers = {
                        ...this._getAuthHeaders(),
                        ...(options.headers || {})
                    };
                    const newResponse = await fetch(url, requestOptions);
                    return newResponse.json();
                }
                
                // Другие ошибки авторизации
                throw new Error(errorData.error || 'Ошибка авторизации');
            }
            
            if (!response.ok) {
                try {
                    // Пробуем получить JSON с ошибкой
                    const errorData = await response.json();
                    throw new Error(errorData.error || `Ошибка запроса: ${response.status}`);
                } catch (jsonError) {
                    // Если не удалось распарсить JSON (например, получили HTML), получаем текст
                    const errorText = await response.text();
                    console.error(`Ошибка HTTP ${response.status}, ответ:`, errorText.substring(0, 200));
                    throw new Error(`Ошибка запроса: ${response.status}. Сервер вернул не-JSON ответ.`);
                }
            }
            
            try {
                return await response.json();
            } catch (jsonError) {
                console.error('Ошибка при парсинге JSON ответа:', jsonError);
                throw new Error('Некорректный формат ответа от сервера (не JSON)');
            }
        } catch (error) {
            console.error('Ошибка запроса:', error);
            throw error;
        }
    }

    /**
     * Возвращает заголовки для аутентифицированных запросов
     * @returns {Object} Заголовки авторизации
     */
    _getAuthHeaders() {
        const headers = {
            'Content-Type': 'application/json'
        };

        if (this.csrfToken) {
            headers['X-CSRF-Token'] = this.csrfToken;
            // Добавляем отладочный вывод
            // console.log('Добавлен CSRF токен в заголовки:', this.csrfToken);
        } else {
            console.warn('CSRF токен отсутствует в заголовках!');
        }

        return headers;
    }

    /**
     * Устанавливает данные аутентификации
     * @param {Object} data Данные аутентификации
     * @private
     */
    _setAuthData(data) {
        this.accessToken = null; // Всегда null в cookie режиме для ясности

        this.csrfToken = data.csrf_token;
        this.user = data.user;
        
        // ИЗМЕНЕНО: Логика определения времени истечения
        let expiryTimestampMs = null;
        const defaultExpiryMinutes = 15; // Стандартное время жизни access token

        // Пытаемся получить время жизни из ответа (если бэкенд его добавит)
        if (data.access_token_expires_at) { // Абсолютное время в секундах
           expiryTimestampMs = data.access_token_expires_at * 1000;
        } else if (data.access_token_expires_in) { // Длительность в секундах
            expiryTimestampMs = Date.now() + data.access_token_expires_in * 1000;
        } else {
            // Fallback: Используем стандартное время жизни access token
            console.warn(`Expiry information not provided by backend. Using default ${defaultExpiryMinutes} minutes.`);
            expiryTimestampMs = Date.now() + defaultExpiryMinutes * 60 * 1000;
        }

        if (expiryTimestampMs && !isNaN(expiryTimestampMs)) {
             this.tokenExpiry = new Date(expiryTimestampMs);
             // Сохраняем время истечения токена (оценка)
             localStorage.setItem('token_expiry', this.tokenExpiry.toISOString());
        } else {
             console.error('Failed to calculate a valid token expiry date.');
             // Устанавливаем короткий fallback, чтобы избежать ошибок
             this.tokenExpiry = new Date(Date.now() + 60 * 1000); // 1 минута
             localStorage.removeItem('token_expiry');
        }
        
        // УДАЛЕНО: Не сохраняем access_token в localStorage
        // localStorage.setItem('access_token', data.access_token);
        localStorage.removeItem('access_token'); // Очищаем старое значение на всякий случай
        
        // Сохраняем CSRF токен отдельно
        localStorage.setItem('csrf_token', data.csrf_token);
        
        // Сохраняем данные пользователя
        localStorage.setItem('user', JSON.stringify(data.user));
        
        // УДАЛЕНО: Не сохраняем token_expiry напрямую здесь, уже сохранили выше
        // localStorage.setItem('token_expiry', expiryDate.toISOString());
        
        // Сохраняем ID текущей сессии
        if (data.current_session_id) {
            this.currentSessionId = data.current_session_id;
            localStorage.setItem('session_id', data.current_session_id);
        }
        
        // ИЗМЕНЕНО: Определяем режим аутентификации - если есть CSRF токен, значит режим cookie
        this.useCookieAuth = !!this.csrfToken;
        // Сохраняем режим аутентификации в localStorage
        localStorage.setItem('use_cookie_auth', this.useCookieAuth.toString());
        console.log('Режим аутентификации установлен:', this.useCookieAuth ? 'Cookie Auth' : 'Bearer Auth',
                  'и сохранен в localStorage');
    }

    /**
     * Очищает данные аутентификации
     * @private
     */
    _clearAuthData() {
        this.accessToken = null;
        this.csrfToken = null;
        this.tokenExpiry = null;
        this.user = null;
        this.currentSessionId = null;
        this.initialized = false;
        
        // Очищаем таймер проверки токена
        if (this.tokenCheckInterval) {
            clearInterval(this.tokenCheckInterval);
            this.tokenCheckInterval = null;
        }
        
        // Удаляем данные из localStorage
        localStorage.removeItem('access_token');
        localStorage.removeItem('csrf_token');
        localStorage.removeItem('token_expiry');
        localStorage.removeItem('user');
        localStorage.removeItem('session_id');
        
        // Нет нужды удалять httpOnly куки на клиенте, они очищаются на сервере
    }

    /**
     * Настраивает обработчики событий
     */
    _setupEventListeners() {
        // Обработка WebSocket-событий
        window.addEventListener('ws_logout', () => {
            console.log('Получено событие выхода через WebSocket');
            this.logout();
        });

        window.addEventListener('ws_password_changed', () => {
            console.log('Получено событие смены пароля через WebSocket');
            alert('Ваш пароль был изменен. Пожалуйста, войдите снова.');
            this.logout();
        });

        window.addEventListener('ws_session_revoked', (event) => {
            console.log('Получено событие отзыва сессии через WebSocket', event.detail);
            if (event.detail && event.detail.session_id === this.currentSessionId) {
                alert('Ваша сессия была завершена на другом устройстве.');
                this.logout();
            } else {
                // Если это не текущая сессия, просто обновляем список сессий
                if (window.location.pathname.includes('sessions.html')) {
                    loadSessions();
                }
            }
        });
    }

    /**
     * Запускает проверку срока действия токена
     */
    _startTokenCheck() {
        this._stopTokenCheck(); // Останавливаем текущий таймер, если есть

        // Проверяем токен каждую минуту
        this.tokenCheckInterval = setInterval(() => {
            if (!this.tokenExpiry) {
                this._stopTokenCheck();
                return;
            }

            const now = new Date();
            const tokenExpiryTime = new Date(this.tokenExpiry);
            
            // Если осталось меньше 5 минут до истечения, обновляем токен
            if (tokenExpiryTime - now < 5 * 60 * 1000) {
                console.log('Автоматическое обновление токена (осталось менее 5 минут)');
                this.refreshToken()
                    .catch(error => console.error('Ошибка при автоматическом обновлении токена:', error));
            }
        }, 60000); // Проверка каждую минуту
    }

    /**
     * Останавливает проверку срока действия токена
     */
    _stopTokenCheck() {
        if (this.tokenCheckInterval) {
            clearInterval(this.tokenCheckInterval);
            this.tokenCheckInterval = null;
        }
    }

    /**
     * Подключает WebSocket для получения уведомлений
     */
    async _connectWebSocket() {
        // Проверяем статус аутентификации
        if (this.ws) {
            console.log('WebSocket уже подключен, пропускаем');
            return;
        }
        
        if (!this.isAuthenticated()) {
            console.error('Невозможно подключить WebSocket: пользователь не аутентифицирован');
            console.log('Состояние аутентификации:', {
                useCookieAuth: this.useCookieAuth,
                hasAccessToken: !!this.accessToken,
                hasCsrfToken: !!this.csrfToken,
                hasUser: !!this.user,
                tokenExpiry: this.tokenExpiry
            });
            return;
        }

        console.log('Попытка подключения WebSocket в режиме:', this.useCookieAuth ? 'Cookie Auth' : 'Bearer Auth');

        try {
            let wsToken;
            let enforceBearer = false; // Флаг для принудительного Bearer режима
            
            // Если у нас есть access token, всегда имеем fallback
            const hasAccessTokenFallback = !!this.accessToken;
            
            // В зависимости от режима авторизации получаем токен для WebSocket
            if (this.useCookieAuth) {
                try {
                    // В режиме Cookie Auth получаем специальный временный тикет
                    console.log('Запрашиваем WS-тикет через API...');
                    wsToken = await this.getWsTicket();
                    console.log('Получен WebSocket тикет для подключения:', wsToken ? 'успешно' : 'ошибка (пустой тикет)');
                    
                    // Проверяем, не использовался ли fallback на access_token
                    // Если wsToken === this.accessToken, значит, метод вернул fallback
                    if (wsToken === this.accessToken) {
                        console.log('getWsTicket вернул fallback. Переключаемся на режим Bearer Auth');
                        enforceBearer = true;
                    }
                } catch (ticketError) {
                    console.error('Ошибка при получении WebSocket тикета:', ticketError);
                    
                    // Если у нас есть access_token как fallback, используем его
                    if (hasAccessTokenFallback) {
                        console.log('Переключаемся на Bearer Auth после ошибки получения WS-тикета');
                        wsToken = this.accessToken;
                        enforceBearer = true;
                    } else {
                        this._showConnectionWarning();
                        return;
                    }
                }
            } else {
                // В режиме Bearer Auth используем access_token
                wsToken = this.accessToken;
                console.log('Используем access_token для WebSocket:', wsToken ? 'доступен' : 'отсутствует');
            }
            
            // Проверяем, что получили токен
            if (!wsToken) {
                console.error('Отсутствует токен для WebSocket подключения');
                this._showConnectionWarning();
                return;
            }
            
            // Если мы вынуждены использовать Bearer Auth (из-за отсутствия эндпоинта WS-тикета)
            if (enforceBearer && this.useCookieAuth) {
                console.log('Временно переключаемся на режим Bearer для WebSocket соединения');
                // Не меняем this.useCookieAuth глобально, так как это только для WS
            }

            // Создаем WebSocket соединение
            console.log(`Подключение к WebSocket: ws://${window.location.host}/ws с токеном (скрыт)`);
            
            // Проверяем еще раз, что токен не undefined и не null
            if (!wsToken || wsToken === 'undefined' || wsToken === 'null') {
                console.error('КРИТИЧЕСКАЯ ОШИБКА: wsToken был определен, но имеет значение undefined или null');
                this._showConnectionWarning();
                throw new Error('Неверный WebSocket токен: ' + (wsToken === undefined ? 'undefined' : wsToken));
            }
            
            this.ws = new WebSocket(`ws://${window.location.host}/ws?token=${wsToken}`);
            
            this.ws.onopen = () => {
                console.log('WebSocket соединение установлено');
                // Сбрасываем счетчик попыток переподключения
                this.wsReconnectAttempts = 0;
                
                // Если было отображено предупреждение, скрываем его
                if (this.wsDisconnected) {
                    this._hideConnectionWarning();
                    this.wsDisconnected = false;
                }
            };
            
            this.ws.onmessage = (event) => {
                try {
                    const data = JSON.parse(event.data);
                    console.log('WebSocket сообщение:', data);
                    
                    // Обрабатываем события
                    if (data.event === 'session_revoked') {
                        const customEvent = new CustomEvent('ws_session_revoked', { 
                            detail: data 
                        });
                        window.dispatchEvent(customEvent);
                    } else if (data.event === 'logout_all_devices') {
                        window.dispatchEvent(new Event('ws_logout'));
                    } else if (data.event === 'session_limit_updated') {
                        // Обновляем лимит сессий
                        if (data.limit) {
                            this.sessionLimit = data.limit;
                            localStorage.setItem('session_limit', this.sessionLimit.toString());
                        }
                    }
                } catch (error) {
                    console.error('Ошибка при обработке WebSocket сообщения:', error);
                }
            };
            
            this.ws.onerror = (error) => {
                console.error('WebSocket ошибка:', error);
                this._showConnectionWarning();
            };
            
            this.ws.onclose = (event) => {
                console.log('WebSocket соединение закрыто:', event.code, event.reason);
                this.ws = null;
                
                // Показываем предупреждение о потере соединения
                this._showConnectionWarning();
                this.wsDisconnected = true;
                
                // Если код 4001 - ошибка аутентификации WebSocket
                if (event.code === 4001) {
                    console.log('Ошибка аутентификации WebSocket (код 4001)');
                    
                    if (this.useCookieAuth) {
                        // В режиме Cookie Auth нужно получить новый тикет
                        console.log('Режим Cookie Auth: требуется получить новый WebSocket тикет.');
                    } else {
                        // В режиме Bearer Auth, пробуем обновить токен
                        console.log('Режим Bearer Auth: пробуем обновить токен и переподключиться.');
                        this.refreshToken()
                            .then(() => {
                                console.log('Токен успешно обновлен, переподключаемся к WebSocket...');
                                this._connectWebSocket();
                            })
                            .catch(error => {
                                console.error('Ошибка обновления токена для WebSocket:', error);
                            });
                        return; // Выходим, так как переподключение будет после обновления токена
                    }
                }
                
                // Для других кодов ошибок или после ошибки 4001 в режиме Cookie Auth
                // Пробуем переподключиться, если мы все еще авторизованы
                if (this.isAuthenticated() && this.wsReconnectAttempts < this.wsMaxReconnectAttempts) {
                    this.wsReconnectAttempts++;
                    const reconnectDelay = Math.min(1000 * Math.pow(2, this.wsReconnectAttempts - 1), 30000);
                    console.log(`Переподключение к WebSocket через ${reconnectDelay}ms (попытка ${this.wsReconnectAttempts}/${this.wsMaxReconnectAttempts})`);
                    setTimeout(() => this._connectWebSocket(), reconnectDelay);
                } else if (this.wsReconnectAttempts >= this.wsMaxReconnectAttempts) {
                    console.error(`Достигнут лимит попыток переподключения (${this.wsMaxReconnectAttempts}). Больше попыток не будет.`);
                }
            };
        } catch (error) {
            console.error('Ошибка при подключении WebSocket:', error);
            this._showConnectionWarning();
        }
    }

    /**
     * Отключает WebSocket соединение
     */
    _disconnectWebSocket() {
        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }
    }

    /**
     * Отображает предупреждение о потере соединения с WebSocket
     */
    _showConnectionWarning() {
        // Проверяем, не отображено ли уже предупреждение
        if (document.getElementById('ws-connection-warning')) {
            return;
        }
        
        // Создаем элемент предупреждения
        const warning = document.createElement('div');
        warning.id = 'ws-connection-warning';
        warning.className = 'connection-warning';
        warning.innerHTML = `
            <div class="warning-icon">⚠️</div>
            <div class="warning-message">
                <strong>Соединение потеряно</strong>
                <p>Уведомления и статус сессии могут не работать корректно. Пытаемся восстановить соединение...</p>
            </div>
        `;
        
        // Добавляем стили если их нет
        if (!document.getElementById('connection-warning-styles')) {
            const style = document.createElement('style');
            style.id = 'connection-warning-styles';
            style.textContent = `
                .connection-warning {
                    position: fixed;
                    top: 10px;
                    right: 10px;
                    background-color: #fff3cd;
                    border: 1px solid #ffeeba;
                    border-left: 4px solid #ffc107;
                    color: #856404;
                    padding: 12px;
                    border-radius: 4px;
                    box-shadow: 0 2px 4px rgba(0,0,0,0.1);
                    z-index: 9999;
                    display: flex;
                    align-items: flex-start;
                    max-width: 400px;
                }
                .warning-icon {
                    font-size: 20px;
                    margin-right: 10px;
                }
                .warning-message strong {
                    display: block;
                    margin-bottom: 5px;
                }
                .warning-message p {
                    margin: 0;
                    font-size: 14px;
                }
            `;
            document.head.appendChild(style);
        }
        
        // Добавляем на страницу
        document.body.appendChild(warning);
    }

    /**
     * Скрывает предупреждение о потере соединения
     */
    _hideConnectionWarning() {
        const warning = document.getElementById('ws-connection-warning');
        if (warning) {
            warning.remove();
        }
    }

    /**
     * Показывает предупреждение о достижении лимита сессий
     */
    _showSessionLimitWarning() {
        // Проверяем, отображается ли уже предупреждение
        if (document.getElementById('session-limit-warning')) {
            return;
        }
        
        // Создаем уведомление о лимите сессий
        const notification = document.createElement('div');
        notification.id = 'session-limit-warning';
        notification.className = 'notification session-limit-warning';
        notification.innerHTML = `
            <div class="notification-content">
                <div class="notification-icon">⚠️</div>
                <div class="notification-message">
                    <strong>Предупреждение: достигнут лимит сессий</strong>
                    <p>Вы достигли максимального количества активных сессий (${this.sessionLimit}). 
                    Рекомендуется завершить неиспользуемые сессии.</p>
                </div>
                <button class="notification-close">&times;</button>
            </div>
        `;
        
        // Добавляем стили, если их еще нет
        if (!document.getElementById('session-limit-styles')) {
            const style = document.createElement('style');
            style.id = 'session-limit-styles';
            style.textContent = `
                .session-limit-warning {
                    position: fixed;
                    bottom: 20px;
                    right: 20px;
                    background-color: #fff3cd;
                    border: 1px solid #ffeeba;
                    border-left: 4px solid #ffc107;
                    color: #856404;
                    padding: 0;
                    border-radius: 4px;
                    box-shadow: 0 2px 8px rgba(0,0,0,0.15);
                    z-index: 9999;
                    max-width: 400px;
                    animation: slideIn 0.3s ease-out;
                }
                .notification-content {
                    display: flex;
                    padding: 12px;
                }
                .notification-icon {
                    font-size: 24px;
                    margin-right: 10px;
                }
                .notification-message {
                    flex: 1;
                }
                .notification-message strong {
                    display: block;
                    margin-bottom: 5px;
                }
                .notification-message p {
                    margin: 0;
                    font-size: 14px;
                }
                .notification-close {
                    background: none;
                    border: none;
                    font-size: 20px;
                    cursor: pointer;
                    color: #856404;
                    margin-left: 10px;
                }
                @keyframes slideIn {
                    from { transform: translateX(100%); opacity: 0; }
                    to { transform: translateX(0); opacity: 1; }
                }
            `;
            document.head.appendChild(style);
        }
        
        // Добавляем на страницу
        document.body.appendChild(notification);
        
        // Добавляем обработчик для кнопки закрытия
        const closeButton = notification.querySelector('.notification-close');
        if (closeButton) {
            closeButton.addEventListener('click', () => {
                notification.remove();
            });
        }
        
        // Автоматически скрываем через 10 секунд
        setTimeout(() => {
            if (notification.parentNode) {
                notification.remove();
            }
        }, 10000);
    }
}

// Экспортируем синглтон
const authService = new AuthService();
export default authService; 