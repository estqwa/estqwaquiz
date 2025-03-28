// В начале файла добавляем импорт AuthService
import authService from './authService.js';

// Основные настройки
const API_BASE_URL = '/api';
let authToken = null;
let currentUser = null;
let currentQuiz = null;

// DOM элементы
const loginSection = document.getElementById('login-section');
const adminPanel = document.getElementById('admin-panel');
const quizzesSection = document.getElementById('quizzes-section');
const createQuizSection = document.getElementById('create-quiz-section');
const quizDetailsSection = document.getElementById('quiz-details-section');
const resetPasswordSection = document.getElementById('reset-password-section');
const addQuestionsForm = document.getElementById('add-questions-form');
const scheduleQuizForm = document.getElementById('schedule-quiz-form');
const restartQuizForm = document.getElementById('restart-quiz-form');
const quizResultsSection = document.getElementById('quiz-results-section');
const userInfoElement = document.getElementById('username');
const quizzesList = document.getElementById('quizzes-list');
const quizInfoElement = document.getElementById('quiz-info');
const questionItemsElement = document.getElementById('question-items');
const quizResultsElement = document.getElementById('quiz-results');
const loginError = document.getElementById('login-error');
const resetResult = document.getElementById('reset-result');
const toast = document.getElementById('toast');

// Обработчики событий
const loginForm = document.getElementById('login-form');
if (loginForm) loginForm.addEventListener('submit', handleLogin);

const logoutBtn = document.getElementById('logout-btn');
if (logoutBtn) logoutBtn.addEventListener('click', handleLogout);

const viewQuizzesBtn = document.getElementById('view-quizzes-btn');
if (viewQuizzesBtn) viewQuizzesBtn.addEventListener('click', () => switchSection(quizzesSection));

const createQuizBtn = document.getElementById('create-quiz-btn');
if (createQuizBtn) createQuizBtn.addEventListener('click', () => switchSection(createQuizSection));

const resetPasswordBtn = document.getElementById('reset-password-btn');
if (resetPasswordBtn) resetPasswordBtn.addEventListener('click', () => switchSection(resetPasswordSection));

const resetPasswordForm = document.getElementById('reset-password-form');
if (resetPasswordForm) resetPasswordForm.addEventListener('submit', handleResetPassword);

const createQuizForm = document.getElementById('create-quiz-form');
if (createQuizForm) createQuizForm.addEventListener('submit', handleCreateQuiz);

const backToQuizzes = document.getElementById('back-to-quizzes');
if (backToQuizzes) backToQuizzes.addEventListener('click', () => switchSection(quizzesSection));

const backToQuizzesBtn = document.getElementById('back-to-quizzes-btn');
if (backToQuizzesBtn) backToQuizzesBtn.addEventListener('click', () => switchSection(quizzesSection));

const addQuestionsBtn = document.getElementById('add-questions-btn');
if (addQuestionsBtn) addQuestionsBtn.addEventListener('click', showAddQuestionsForm);

const scheduleQuizBtn = document.getElementById('schedule-quiz-btn');
if (scheduleQuizBtn) scheduleQuizBtn.addEventListener('click', showScheduleQuizForm);

const cancelQuizBtn = document.getElementById('cancel-quiz-btn');
if (cancelQuizBtn) cancelQuizBtn.addEventListener('click', handleCancelQuiz);

const restartQuizBtn = document.getElementById('restart-quiz-btn');
if (restartQuizBtn) restartQuizBtn.addEventListener('click', showRestartQuizForm);

const cancelRestartBtn = document.getElementById('cancel-restart');
if (cancelRestartBtn) cancelRestartBtn.addEventListener('click', () => {
    if (restartQuizForm) restartQuizForm.classList.add('hidden');
});

const submitRestartBtn = document.getElementById('submit-restart');
if (submitRestartBtn) submitRestartBtn.addEventListener('click', handleRestartQuiz);

const addMoreQuestionBtn = document.getElementById('add-more-question');
if (addMoreQuestionBtn) addMoreQuestionBtn.addEventListener('click', addQuestionItem);

const submitQuestionsBtn = document.getElementById('submit-questions');
if (submitQuestionsBtn) submitQuestionsBtn.addEventListener('click', handleSubmitQuestions);

const submitScheduleBtn = document.getElementById('submit-schedule');
if (submitScheduleBtn) submitScheduleBtn.addEventListener('click', handleScheduleQuiz);

// Инициализация
initApp();

// Проверка авторизации и отображение нужного блока
async function initApp() {
    // Проверяем параметры URL
    const urlParams = new URLSearchParams(window.location.search);
    const showLogin = urlParams.get('showLogin') === 'true';
    const error = urlParams.get('error');
    
    // Если в URL указан параметр ошибки, показываем сообщение
    if (error === 'session_expired') {
        showToast('Срок действия сессии истек. Пожалуйста, войдите снова.', 'error');
    }
    
    // Инициализируем AuthService
    const authInitResult = await authService.init();

    // Проверяем аутентификацию: подлинный токен имеет приоритет над параметром showLogin
    if (!authService.isAuthenticated()) {
        // Показываем форму входа только если действительно не авторизован
        showLogin();
        
        // Если есть параметр showLogin, удаляем его из URL, чтобы не зацикливаться
        if (showLogin) {
            const newUrl = new URL(window.location);
            newUrl.searchParams.delete('showLogin');
            window.history.replaceState({}, '', newUrl);
        }
        return;
    }

    // Если аутентифицированы, показываем админ-панель,
    // независимо от параметра showLogin
    showAdmin();
    
    // Если в URL был параметр showLogin, удаляем его из истории
    if (showLogin) {
        const newUrl = new URL(window.location);
        newUrl.searchParams.delete('showLogin');
        window.history.replaceState({}, '', newUrl);
    }

    // Если нужен доступ администратора для текущей страницы, проверяем
    if (window.location.href.includes('admin-') && !authService.isAdmin()) {
        alert('Доступ запрещен! Требуются права администратора.');
        window.location.href = 'index.html';
        return;
    }

    // Устанавливаем данные пользователя в интерфейс
    updateUserUI();

    // Инициализируем работу со страницами в зависимости от текущей страницы
    initCurrentPage();
}

// Обновляем функцию для установки пользовательского интерфейса
function updateUserUI() {
    if (!authService.user) return;

    // Обновляем информацию о пользователе в интерфейсе
    const userInfoElements = document.querySelectorAll('.user-info');
    userInfoElements.forEach(el => {
        el.textContent = authService.user.username || authService.user.email;
    });

    // Отображаем/скрываем элементы для администраторов
    const adminElements = document.querySelectorAll('.admin-only');
    adminElements.forEach(el => {
        el.style.display = authService.isAdmin() ? 'block' : 'none';
    });
    
    // Привязываем обработчик к кнопке выхода
    const logoutBtns = document.querySelectorAll('#logout-btn');
    logoutBtns.forEach(btn => {
        btn.addEventListener('click', handleLogout);
    });
}

// Инициализируем страницу в зависимости от её типа
function initCurrentPage() {
    const path = window.location.pathname;
    
    if (path.includes('sessions.html')) {
        // Страница управления сессиями инициализируется в sessions.html
    } else if (path.includes('index.html') || path === '/' || path.endsWith('/admin/')) {
        // Страница дашборда
        loadQuizzes();
    }
}

// Обновляем функцию для выполнения аутентифицированных запросов
async function apiRequest(url, method = 'GET', data = null) {
    try {
        // Формируем объект опций для запроса
        const options = {
            method: method
        };
        
        // Добавляем тело запроса, если есть данные и метод не GET
        if (data && method !== 'GET') {
            options.body = JSON.stringify(data);
        }
        
        console.log(`Выполняем ${method} запрос к ${url}`, options);
        
        // Делаем запрос через authService
        return await authService._authorizedRequest(`${API_BASE_URL}${url}`, options);
    } catch (error) {
        console.error('Ошибка API запроса:', error);
        if (error.message && error.message.includes('Ошибка авторизации')) {
            // Показываем ошибку и перенаправляем на страницу входа при проблемах с аутентификацией
            showToast('Ошибка авторизации. Пожалуйста, войдите снова.', 'error');
            setTimeout(() => {
                authService.logout();
            }, 2000);
        }
        throw error;
    }
}

// Обработчик выхода
async function handleLogout(event) {
    event.preventDefault();
    try {
        await authService.logout();
    } catch (error) {
        console.error('Ошибка при выходе:', error);
        // Насильно редиректим на главную с параметром showLogin
        window.location.href = 'index.html?showLogin=true';
    }
}

// Обработчик входа
async function handleLogin(event) {
    event.preventDefault();
    
    const email = document.getElementById('email').value;
    const password = document.getElementById('password').value;
    
    try {
        await authService.login(email, password);
        // Перенаправление произойдет в AuthService
    } catch (error) {
        console.error('Ошибка входа:', error);
        showToast('Ошибка входа: ' + (error.message || 'Проверьте учетные данные'), 'error');
    }
}

// Загрузка списка викторин
async function loadQuizzes() {
    try {
        const quizzes = await apiRequest('/quizzes', 'GET');
        renderQuizzesList(quizzes);
    } catch (error) {
        showToast('Ошибка при загрузке викторин', 'error');
    }
}

// Отрисовка списка викторин
function renderQuizzesList(quizzes) {
    console.log("Начало отрисовки списка викторин:", quizzes);
    quizzesList.innerHTML = '';
    
    if (quizzes.length === 0) {
        quizzesList.innerHTML = '<p>Викторин пока нет. Создайте вашу первую викторину!</p>';
        return;
    }

    quizzes.forEach(quiz => {
        const quizCard = document.createElement('div');
        quizCard.className = 'quiz-card';
        quizCard.setAttribute('data-quiz-id', quiz.id);
        quizCard.innerHTML = `
            <h3>${quiz.title}</h3>
            <div class="quiz-status status-${quiz.status}">${getStatusText(quiz.status)}</div>
            <p>${quiz.description || 'Без описания'}</p>
            <p><strong>Вопросов:</strong> ${quiz.question_count || 0}</p>
            <p><strong>Дата:</strong> ${formatDate(quiz.scheduled_time)}</p>
        `;
        quizCard.addEventListener('click', function() {
            const quizId = this.getAttribute('data-quiz-id');
            console.log("Клик по карточке викторины с ID:", quizId);
            if (quizId) {
                loadQuizDetails(quizId);
            } else {
                console.error("Не удалось получить ID викторины из атрибута data-quiz-id");
            }
        });
        quizzesList.appendChild(quizCard);
    });
    console.log("Карточки викторин добавлены на страницу, всего:", quizzes.length);
}

// Создание новой викторины
async function handleCreateQuiz(event) {
    event.preventDefault();
    const title = document.getElementById('quiz-title').value;
    const description = document.getElementById('quiz-description').value;
    const scheduledTime = document.getElementById('quiz-scheduled-time').value;

    // Дата должна быть передана, т.к. это обязательное поле на бэкенде
    const quizData = {
        title,
        description,
        scheduled_time: scheduledTime 
            ? new Date(scheduledTime).toISOString() 
            : new Date(Date.now() + 10 * 60 * 1000).toISOString() // Если дата не указана, ставим через 10 минут
    };

    try {
        const result = await apiRequest('/quizzes', 'POST', quizData);
        showToast('Викторина успешно создана');
        
        // Сразу загружаем детали викторины для добавления вопросов
        loadQuizDetails(result.id);
        showToast('Теперь добавьте вопросы к викторине, прежде чем она начнется', 'info');
    } catch (error) {
        showToast('Ошибка при создании викторины: ' + error.message, 'error');
    }
}

// Загрузка подробной информации о викторине
async function loadQuizDetails(quizId) {
    console.log("Загрузка деталей викторины с ID:", quizId);
    try {
        const quiz = await apiRequest(`/quizzes/${quizId}/with-questions`, 'GET');
        console.log("Получены данные викторины:", quiz);
        
        // Убедимся, что у объекта quiz есть свойство questions, даже если оно пустое
        if (!quiz.questions) {
            quiz.questions = [];
            console.log("Добавлено пустое свойство questions, т.к. оно отсутствовало в ответе API");
        }
        
        currentQuiz = quiz;
        
        // Сначала переключаем секцию на детали викторины
        console.log("Переключение секции на детали викторины");
        
        // Явно скрываем все секции, кроме деталей викторины
        if (quizzesSection) quizzesSection.classList.add('hidden');
        if (createQuizSection) createQuizSection.classList.add('hidden');
        if (quizDetailsSection) {
            quizDetailsSection.classList.remove('hidden');
            console.log("Секция деталей викторины показана");
        } else {
            console.error("Секция деталей викторины не найдена!");
        }
        
        // Затем отображаем детали
        renderQuizDetails(quiz);
        
        // Проверяем, что секция действительно видима
        if (quizDetailsSection && quizDetailsSection.classList.contains('hidden')) {
            console.warn("Секция деталей все еще скрыта после переключения, принудительно показываем");
            quizDetailsSection.classList.remove('hidden');
        }
    } catch (error) {
        console.error("Ошибка при загрузке деталей викторины:", error);
        showToast('Ошибка при загрузке информации о викторине: ' + error.message, 'error');
    }
}

// Отрисовка информации о викторине
function renderQuizDetails(quiz) {
    console.log("Начало отрисовки деталей викторины:", quiz);
    
    // Убедимся, что у викторины есть массив вопросов
    if (!quiz.questions) {
        quiz.questions = [];
        console.log("Инициализирован пустой массив вопросов в renderQuizDetails");
    }
    
    // Проверим секцию деталей викторины
    console.log("Секция деталей видима?", !quizDetailsSection.classList.contains('hidden'));
    
    document.getElementById('quiz-details-title').textContent = quiz.title;
    
    quizInfoElement.innerHTML = `
        <div class="quiz-info-item"><strong>Статус:</strong> <span class="quiz-status status-${quiz.status}">${getStatusText(quiz.status)}</span></div>
        <div class="quiz-info-item"><strong>Описание:</strong> ${quiz.description || 'Нет описания'}</div>
        <div class="quiz-info-item"><strong>Запланировано на:</strong> ${formatDate(quiz.scheduled_time) || 'Не запланировано'}</div>
        <div class="quiz-info-item"><strong>Количество вопросов:</strong> ${quiz.questions.length}</div>
    `;

    // Отображение вопросов
    console.log("Вопросы для отрисовки:", quiz.questions);
    renderQuizQuestions(quiz.questions);

    // Показать/скрыть кнопки в зависимости от статуса
    const canSchedule = ['created', 'cancelled'].includes(quiz.status);
    const canCancel = ['scheduled'].includes(quiz.status);
    // Изменяем логику: вопросы теперь можно добавлять для созданных и запланированных викторин
    const canAddQuestions = ['created', 'scheduled'].includes(quiz.status);
    // Перезапуск доступен для викторин в любом статусе, если у них есть вопросы
    const canRestart = quiz.questions && quiz.questions.length > 0;
    
    // Возможность планирования викторины только если у нее есть вопросы
    const hasQuestions = quiz.questions && quiz.questions.length > 0;
    const actualCanSchedule = canSchedule && hasQuestions;
    
    console.log("Видимость кнопок - Запланировать:", actualCanSchedule, "Отменить:", canCancel, "Добавить вопросы:", canAddQuestions, "Перезапустить:", canRestart);
    
    document.getElementById('schedule-quiz-btn').style.display = actualCanSchedule ? 'inline-block' : 'none';
    document.getElementById('cancel-quiz-btn').style.display = canCancel ? 'inline-block' : 'none';
    document.getElementById('add-questions-btn').style.display = canAddQuestions ? 'inline-block' : 'none';
    document.getElementById('restart-quiz-btn').style.display = canRestart ? 'inline-block' : 'none';

    // Обновляем подсказки для пользователя
    if (canSchedule && !hasQuestions) {
        showToast('Сначала нужно добавить вопросы к викторине, прежде чем её можно будет запланировать', 'info');
    } else if (quiz.status === 'scheduled' && (!quiz.questions || quiz.questions.length === 0)) {
        // Особое предупреждение для запланированных викторин без вопросов
        showToast('Внимание! У этой запланированной викторины нет вопросов. Добавьте вопросы перед началом викторины.', 'warning');
    }

    // Загрузить результаты, если викторина завершена
    if (quiz.status === 'completed') {
        loadQuizResults(quiz.id);
    } else {
        quizResultsSection.classList.add('hidden');
    }
    
    console.log("Завершение отрисовки деталей викторины");
}

// Отрисовка вопросов викторины
function renderQuizQuestions(questions) {
    console.log("Отрисовка вопросов, количество:", questions ? questions.length : 0);
    
    // Очистка обоих контейнеров (для обратной совместимости и на случай если оба используются)
    const questionItemsElement = document.getElementById('question-items');
    const quizQuestionsElement = document.getElementById('quiz-questions');
    
    if (questionItemsElement) questionItemsElement.innerHTML = '';
    if (quizQuestionsElement) quizQuestionsElement.innerHTML = '';
    
    // Убедимся, что questions - это массив
    if (!questions || !Array.isArray(questions) || questions.length === 0) {
        const message = '<p>Вопросы еще не добавлены</p>';
        if (questionItemsElement) questionItemsElement.innerHTML = message;
        if (quizQuestionsElement) quizQuestionsElement.innerHTML = message;
        console.log("Нет вопросов для отображения");
        return;
    }

    questions.forEach((question, index) => {
        console.log(`Отрисовка вопроса ${index + 1}:`, question);
        
        // Конвертируем options из возможного формата объектов в строки для отображения
        const optionsArray = Array.isArray(question.options) 
            ? question.options.map(opt => {
                if (typeof opt === 'string') return opt;
                if (typeof opt === 'object' && opt !== null && 'text' in opt) return opt.text;
                return String(opt);
              })
            : [];
            
        const questionEl = document.createElement('div');
        questionEl.className = 'question-details';
        questionEl.innerHTML = `
            <h4>Вопрос ${index + 1}</h4>
            <p><strong>Текст:</strong> ${question.text || 'Нет текста'}</p>
            <p><strong>Варианты:</strong></p>
            <ol>
                ${optionsArray.map(option => `<li>${option}</li>`).join('')}
            </ol>
            <p><strong>Правильный ответ:</strong> ${optionsArray[question.correct_option] || 'Не указан'}</p>
            <p><strong>Время на ответ:</strong> ${question.time_limit_sec || 0} сек.</p>
            <p><strong>Очки:</strong> ${question.point_value || 0}</p>
        `;
        
        // Добавляем в оба контейнера, если они существуют
        if (questionItemsElement) questionItemsElement.appendChild(questionEl.cloneNode(true));
        if (quizQuestionsElement) quizQuestionsElement.appendChild(questionEl);
        
        console.log(`Вопрос ${index + 1} отрисован`);
    });
    
    console.log("Все вопросы отрисованы успешно");
}

// Загрузка результатов викторины
async function loadQuizResults(quizId) {
    try {
        const results = await apiRequest(`/quizzes/${quizId}/results`, 'GET');
        renderQuizResults(results);
        quizResultsSection.classList.remove('hidden');
    } catch (error) {
        showToast('Ошибка при загрузке результатов', 'error');
    }
}

// Отрисовка результатов викторины
function renderQuizResults(results) {
    if (!results || results.length === 0) {
        quizResultsElement.innerHTML = '<p>Результаты пока недоступны.</p>';
        return;
    }

    let tableHTML = `
        <table class="results-table">
            <thead>
                <tr>
                    <th>Ранг</th>
                    <th>Пользователь</th>
                    <th>Очки</th>
                    <th>Правильных ответов</th>
                </tr>
            </thead>
            <tbody>
    `;

    results.forEach(result => {
        tableHTML += `
            <tr>
                <td>${result.rank}</td>
                <td>${result.username}</td>
                <td>${result.score}</td>
                <td>${result.correct_answers}</td>
            </tr>
        `;
    });

    tableHTML += `
            </tbody>
        </table>
    `;

    quizResultsElement.innerHTML = tableHTML;
}

// Показать форму добавления вопросов
function showAddQuestionsForm() {
    // Проверяем, что викторина в подходящем статусе для добавления вопросов
    if (!currentQuiz || !(['created', 'scheduled'].includes(currentQuiz.status))) {
        showToast('Вопросы можно добавлять только к созданным или запланированным викторинам.', 'error');
        return;
    }
    
    addQuestionsForm.classList.remove('hidden');
    const questionsContainer = document.getElementById('questions-container');
    questionsContainer.innerHTML = '';
    addQuestionItem();
}

// Добавление элемента вопроса в форму
function addQuestionItem() {
    const questionsContainer = document.getElementById('questions-container');
    const questionItem = document.createElement('div');
    questionItem.className = 'question-item';
    questionItem.innerHTML = `
        <div class="form-group">
            <label>Текст вопроса:</label>
            <input type="text" class="question-text" required>
        </div>
        <div class="form-group">
            <label>Варианты ответов (по одному в строке):</label>
            <textarea class="question-options" rows="4" required></textarea>
        </div>
        <div class="form-group">
            <label>Правильный вариант (начиная с 0):</label>
            <input type="number" class="question-correct-option" min="0" required>
        </div>
        <div class="form-group">
            <label>Время на ответ (сек):</label>
            <input type="number" class="question-time-limit" min="5" value="15" required>
        </div>
        <div class="form-group">
            <label>Количество очков:</label>
            <input type="number" class="question-points" min="1" value="10" required>
        </div>
        <button type="button" class="btn btn-danger remove-question">Удалить</button>
    `;
    
    questionItem.querySelector('.remove-question').addEventListener('click', function() {
        if (document.querySelectorAll('.question-item').length > 1) {
            questionItem.remove();
        } else {
            showToast('Должен быть хотя бы один вопрос', 'error');
        }
    });

    questionsContainer.appendChild(questionItem);
}

// Отправка вопросов
async function handleSubmitQuestions() {
    if (!currentQuiz) return;

    // Проверяем, что викторина в допустимом статусе
    if (!['created', 'scheduled'].includes(currentQuiz.status)) {
        showToast('Вопросы можно добавлять только к созданным или запланированным викторинам.', 'error');
        addQuestionsForm.classList.add('hidden');
        return;
    }

    const questions = [];
    const questionItems = document.querySelectorAll('.question-item');
    let hasErrors = false;

    questionItems.forEach((item, index) => {
        const text = item.querySelector('.question-text').value.trim();
        const optionsText = item.querySelector('.question-options').value;
        const correctOption = parseInt(item.querySelector('.question-correct-option').value);
        const timeLimit = parseInt(item.querySelector('.question-time-limit').value);
        const pointValue = parseInt(item.querySelector('.question-points').value);

        // Разбиваем текст на отдельные строки для вариантов и удаляем пустые строки
        // ВАЖНО: Мы используем ИМЕННО массив JavaScript, который будет корректно сериализован в JSON
        // и правильно воспринят бэкендом для хранения в PostgreSQL JSONB
        const options = optionsText.split('\n').filter(line => line.trim() !== '');
        
        // Проверка на пустой текст вопроса
        if (!text) {
            showToast(`Ошибка в вопросе ${index + 1}: текст вопроса не может быть пустым`, 'error');
            hasErrors = true;
            return;
        }
        
        // Проверка на минимальное количество вариантов (2)
        if (options.length < 2) {
            showToast(`Ошибка в вопросе ${index + 1}: необходимо минимум 2 варианта ответа`, 'error');
            hasErrors = true;
            return;
        }
        
        // Проверка на корректный номер правильного ответа
        if (isNaN(correctOption) || correctOption < 0 || correctOption >= options.length) {
            showToast(`Ошибка в вопросе ${index + 1}: некорректный номер правильного ответа`, 'error');
            hasErrors = true;
            return;
        }
        
        // Если все проверки пройдены, добавляем вопрос
        const questionObj = {
            text: text,
            options: options,  // НЕ преобразуем в строку, отправляем как массив
            correct_option: correctOption,
            time_limit_sec: timeLimit,
            point_value: pointValue
        };
        questions.push(questionObj);
    });

    if (hasErrors) {
        return;
    }

    if (questions.length === 0) {
        showToast('Пожалуйста, заполните все поля вопросов корректно', 'error');
        return;
    }

    console.log('Вопросы для отправки:', questions);

    try {
        // Отправляем запрос на сервер с модифицированными данными
        const result = await apiRequest(`/quizzes/${currentQuiz.id}/questions`, 'POST', {
            questions: questions
        });
        
        console.log('Успешный ответ:', result);
        showToast('Вопросы успешно добавлены');
        addQuestionsForm.classList.add('hidden');
        
        // Повторно загружаем детали викторины, чтобы отобразить добавленные вопросы
        loadQuizDetails(currentQuiz.id);
        
        // Если викторина в статусе created и у нас теперь есть вопросы, показываем подсказку
        if (currentQuiz.status === 'created') {
            showToast('Теперь вы можете запланировать викторину на удобное время', 'info');
        }
    } catch (error) {
        console.error('Ошибка при добавлении вопросов:', error);
        let errorMessage = error.message || 'Неизвестная ошибка';
        
        // Проверяем на наличие детальной информации об ошибке
        if (typeof errorMessage === 'string' && errorMessage.includes('pq:')) {
            // Ошибка PostgreSQL - извлекаем основное сообщение
            const match = errorMessage.match(/pq:([^"]+)/);
            if (match && match[1]) {
                errorMessage = match[1].trim();
            }
        }
        
        showToast('Ошибка: ' + errorMessage, 'error');
    }
}

// Показать форму планирования времени
function showScheduleQuizForm() {
    scheduleQuizForm.classList.remove('hidden');
    const now = new Date();
    now.setMinutes(now.getMinutes() + 10); // Минимум через 10 минут
    const formattedDateTime = now.toISOString().slice(0, 16);
    document.getElementById('schedule-time').value = formattedDateTime;
    document.getElementById('schedule-time').min = formattedDateTime;
}

// Планировать викторину
async function handleScheduleQuiz() {
    if (!currentQuiz) return;

    const scheduledTime = document.getElementById('schedule-time').value;
    if (!scheduledTime) {
        showToast('Выберите время проведения викторины', 'error');
        return;
    }

    // Проверка наличия вопросов
    if (!currentQuiz.questions || currentQuiz.questions.length === 0) {
        showToast('Нельзя запланировать викторину без вопросов. Сначала добавьте вопросы.', 'error');
        scheduleQuizForm.classList.add('hidden');
        return;
    }

    try {
        await apiRequest(`/quizzes/${currentQuiz.id}/schedule`, 'PUT', {
            scheduled_time: new Date(scheduledTime).toISOString()
        });
        showToast('Викторина успешно запланирована');
        scheduleQuizForm.classList.add('hidden');
        loadQuizDetails(currentQuiz.id);
    } catch (error) {
        console.error('Ошибка при планировании викторины:', error);
        showToast(`Ошибка при планировании викторины: ${error.message}`, 'error');
        scheduleQuizForm.classList.add('hidden');
    }
}

// Отмена викторины
async function handleCancelQuiz() {
    if (!currentQuiz) return;

    if (!confirm('Вы уверены, что хотите отменить эту викторину?')) {
        return;
    }

    try {
        await apiRequest(`/quizzes/${currentQuiz.id}/cancel`, 'PUT', null);
        showToast('Викторина отменена');
        loadQuizDetails(currentQuiz.id);
    } catch (error) {
        showToast('Ошибка при отмене викторины', 'error');
    }
}

// Показать форму перезапуска викторины
function showRestartQuizForm() {
    restartQuizForm.classList.remove('hidden');
    const now = new Date();
    now.setMinutes(now.getMinutes() + 10); // Минимум через 10 минут
    const formattedDateTime = now.toISOString().slice(0, 16);
    document.getElementById('restart-time').value = formattedDateTime;
    document.getElementById('restart-time').min = formattedDateTime;
}

// Перезапуск викторины
async function handleRestartQuiz() {
    if (!currentQuiz) return;

    const restartTime = document.getElementById('restart-time').value;
    if (!restartTime) {
        showToast('Выберите новое время проведения викторины', 'error');
        return;
    }

    // Проверка наличия вопросов
    if (!currentQuiz.questions || currentQuiz.questions.length === 0) {
        showToast('Нельзя запланировать викторину без вопросов. Сначала добавьте вопросы.', 'error');
        restartQuizForm.classList.add('hidden');
        return;
    }

    if (!confirm('Вы уверены, что хотите перезапустить эту викторину?')) {
        return;
    }

    try {
        // Перезапуск викторины одним запросом
        // Сервер автоматически изменит статус с "completed" на "scheduled"
        await apiRequest(`/quizzes/${currentQuiz.id}/schedule`, 'PUT', {
            scheduled_time: new Date(restartTime).toISOString()
        });
        
        showToast('Викторина успешно перезапущена на новое время');
        restartQuizForm.classList.add('hidden');
        loadQuizDetails(currentQuiz.id);
    } catch (error) {
        console.error('Ошибка при перезапуске викторины:', error);
        showToast('Ошибка при перезапуске викторины: ' + error.message, 'error');
        restartQuizForm.classList.add('hidden');
    }
}

// Вспомогательные функции
function switchSection(section) {
    console.log("Переключение секции на:", section ? section.id : "неизвестно");
    // Проверяем, что элементы существуют перед обращением к ним
    [quizzesSection, createQuizSection, quizDetailsSection].forEach(s => {
        if (s) {
            s.classList.add('hidden');
            console.log("Скрыта секция:", s.id);
        }
    });
    
    // Скрываем все подформы с проверкой на существование
    if (addQuestionsForm) addQuestionsForm.classList.add('hidden');
    if (scheduleQuizForm) scheduleQuizForm.classList.add('hidden');
    if (restartQuizForm) restartQuizForm.classList.add('hidden');
    
    // Активная вкладка навигации
    document.querySelectorAll('.nav-btn').forEach(btn => btn.classList.remove('active'));
    
    if (section === quizzesSection) {
        const viewQuizzesBtn = document.getElementById('view-quizzes-btn');
        if (viewQuizzesBtn) viewQuizzesBtn.classList.add('active');
        loadQuizzes();
    } else if (section === createQuizSection) {
        const createQuizBtn = document.getElementById('create-quiz-btn');
        if (createQuizBtn) createQuizBtn.classList.add('active');
    }
    
    // Проверяем, что section существует
    if (section) {
        section.classList.remove('hidden');
        console.log("Показана секция:", section.id);
    }
}

function showLogin() {
    if (loginSection) loginSection.classList.remove('hidden');
    if (adminPanel) adminPanel.classList.add('hidden');
    
    // Проверяем, есть ли параметр error в URL
    const urlParams = new URLSearchParams(window.location.search);
    const error = urlParams.get('error');
    
    if (error && loginError) {
        let errorMessage = 'Ошибка входа';
        
        if (error === 'session_expired') {
            errorMessage = 'Срок действия сессии истек. Пожалуйста, войдите снова.';
        } else if (error === 'invalid_credentials') {
            errorMessage = 'Неверный email или пароль. Попробуйте снова.';
        }
        
        loginError.textContent = errorMessage;
        loginError.classList.remove('hidden');
    }
}

function showAdmin() {
    if (loginSection) loginSection.classList.add('hidden');
    if (adminPanel) adminPanel.classList.remove('hidden');
    switchSection(quizzesSection);
}

function formatDate(dateString) {
    if (!dateString) return 'Не указано';
    const date = new Date(dateString);
    return date.toLocaleString('ru-RU', {
        day: '2-digit',
        month: '2-digit',
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
    });
}

function getStatusText(status) {
    const statusMap = {
        'created': 'Создана',
        'scheduled': 'Запланирована',
        'in_progress': 'В процессе',
        'completed': 'Завершена',
        'cancelled': 'Отменена'
    };
    return statusMap[status] || status;
}

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

// Функция для сброса пароля пользователя
async function handleResetPassword(event) {
    event.preventDefault();
    const email = document.getElementById('reset-email').value;
    const newPassword = document.getElementById('reset-new-password').value;

    resetResult.innerHTML = '';
    resetResult.classList.add('hidden');

    try {
        // Отправляем запрос на API для сброса пароля
        const result = await apiRequest('/auth/admin/reset-password', 'POST', { 
            email: email,
            password: newPassword
        });

        // Показываем результат
        resetResult.innerHTML = `
            <div class="success-message">
                <h3>Пароль успешно сброшен</h3>
                <p>Новый пароль для пользователя ${email} установлен.</p>
                <p>Инструкции для пользователя:</p>
                <ol>
                    <li>Перейти на страницу входа</li>
                    <li>Ввести email: ${email}</li>
                    <li>Ввести временный пароль: ${newPassword}</li>
                    <li>После входа рекомендуется сменить пароль в настройках профиля</li>
                </ol>
            </div>
        `;
        resetResult.classList.remove('hidden');
        
        // Очищаем поле email после успешного сброса
        document.getElementById('reset-email').value = '';
        
        showToast('Пароль успешно сброшен', 'success');
    } catch (error) {
        resetResult.innerHTML = `
            <div class="error-message">
                <h3>Ошибка при сбросе пароля</h3>
                <p>${error.message || 'Произошла неизвестная ошибка при сбросе пароля'}</p>
            </div>
        `;
        resetResult.classList.remove('hidden');
        showToast('Ошибка при сбросе пароля', 'error');
    }
}

// Экспортируем важные функции в глобальную область видимости для использования в других скриптах
window.appFunctions = {
    switchSection: switchSection,
    showToast: showToast,
    formatDate: formatDate,
    getStatusText: getStatusText
}; 