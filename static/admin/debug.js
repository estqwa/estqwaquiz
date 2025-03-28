// Отладочные функции для проверки видимости секций и переключения между ними

// Функция для проверки, какие секции сейчас видимы
function checkVisibleSections() {
    const sections = ['login-section', 'admin-panel', 'quizzes-section', 'create-quiz-section', 'quiz-details-section'];
    console.log('Проверка видимости секций:');
    
    sections.forEach(id => {
        const element = document.getElementById(id);
        if (element) {
            console.log(`- ${id}: ${element.classList.contains('hidden') ? 'скрыта' : 'видима'}`);
        } else {
            console.log(`- ${id}: элемент не найден`);
        }
    });
}

// Функция для принудительного отображения секции деталей викторины
function forceShowQuizDetails() {
    console.log('Принудительное отображение деталей викторины:');
    
    const sections = ['quizzes-section', 'create-quiz-section', 'quiz-details-section'];
    sections.forEach(id => {
        const element = document.getElementById(id);
        if (element) {
            if (id === 'quiz-details-section') {
                element.classList.remove('hidden');
                console.log(`- ${id}: сделана видимой`);
            } else {
                element.classList.add('hidden');
                console.log(`- ${id}: скрыта`);
            }
        }
    });
    
    // Проверяем наличие текущей викторины и её содержимое
    if (typeof currentQuiz !== 'undefined' && currentQuiz) {
        console.log('Текущая викторина:', currentQuiz);
    } else {
        console.log('Текущая викторина не определена');
    }
}

// Функция для тестового заполнения деталей викторины
function testRenderQuizDetails() {
    const testQuiz = {
        id: 999,
        title: 'Тестовая викторина',
        description: 'Описание для отладки',
        scheduled_time: new Date().toISOString(),
        status: 'scheduled',
        questions: [],
        question_count: 0
    };
    
    console.log('Тестовая отрисовка деталей викторины:');
    
    // Сначала показываем секцию
    forceShowQuizDetails();
    
    // Затем заполняем её тестовыми данными
    const titleElement = document.getElementById('quiz-details-title');
    if (titleElement) {
        titleElement.textContent = testQuiz.title;
        console.log('- Заголовок викторины обновлен');
    }
    
    const infoElement = document.getElementById('quiz-info');
    if (infoElement) {
        infoElement.innerHTML = `
            <div class="quiz-info-item"><strong>Статус:</strong> <span class="quiz-status status-${testQuiz.status}">Запланирована</span></div>
            <div class="quiz-info-item"><strong>Описание:</strong> ${testQuiz.description}</div>
            <div class="quiz-info-item"><strong>Запланировано на:</strong> ${new Date().toLocaleString('ru-RU')}</div>
            <div class="quiz-info-item"><strong>Количество вопросов:</strong> 0</div>
        `;
        console.log('- Информация о викторине обновлена');
    }
    
    // Отображаем кнопки
    const buttons = {
        'schedule-quiz-btn': false,
        'cancel-quiz-btn': true,
        'add-questions-btn': true,
        'restart-quiz-btn': true
    };
    
    Object.entries(buttons).forEach(([id, visible]) => {
        const button = document.getElementById(id);
        if (button) {
            button.style.display = visible ? 'inline-block' : 'none';
            console.log(`- Кнопка ${id}: ${visible ? 'показана' : 'скрыта'}`);
        }
    });
    
    return 'Тестовая отрисовка выполнена';
}

// Функция для проверки обработчиков событий на карточках викторин
function checkQuizCardHandlers() {
    const quizCards = document.querySelectorAll('.quiz-card');
    console.log(`Найдено ${quizCards.length} карточек викторин`);
    
    quizCards.forEach((card, index) => {
        const quizId = card.getAttribute('data-quiz-id');
        console.log(`Карточка ${index + 1}: ID викторины = ${quizId}`);
        
        // Добавим новый обработчик для тестирования
        card.setAttribute('data-debug-index', index);
        card.addEventListener('click', function() {
            const debugIndex = this.getAttribute('data-debug-index');
            const id = this.getAttribute('data-quiz-id');
            console.log(`Клик по карточке викторины #${debugIndex}, ID = ${id}`);
            
            // Вызываем родные функции
            try {
                if (typeof loadQuizDetails === 'function') {
                    console.log('Вызов loadQuizDetails...');
                    loadQuizDetails(id);
                }
            } catch (e) {
                console.error('Ошибка при вызове loadQuizDetails:', e);
            }
        });
    });
    
    return `Добавлены отладочные обработчики для ${quizCards.length} карточек`;
}

// Функция для проверки структуры HTML
function checkHtmlStructure() {
    const sections = {
        'login-section': 'Секция входа',
        'admin-panel': 'Админ-панель',
        'quizzes-section': 'Список викторин',
        'create-quiz-section': 'Создание викторины',
        'quiz-details-section': 'Детали викторины',
        'questions-list': 'Список вопросов',
        'quiz-questions': 'Контейнер вопросов (quiz-questions)',
        'question-items': 'Контейнер вопросов (question-items)',
        'add-questions-form': 'Форма добавления вопросов',
        'schedule-quiz-form': 'Форма планирования',
        'restart-quiz-form': 'Форма перезапуска',
        'quiz-results-section': 'Секция результатов'
    };
    
    console.log('Проверка структуры HTML:');
    
    Object.entries(sections).forEach(([id, description]) => {
        const element = document.getElementById(id);
        if (element) {
            console.log(`- ${description} (${id}): найден, классы: "${element.className}"`);
        } else {
            console.log(`- ${description} (${id}): НЕ НАЙДЕН`);
        }
    });
}

console.log('Отладочные функции загружены. Доступны:');
console.log('- checkVisibleSections() - Проверка видимости секций');
console.log('- forceShowQuizDetails() - Принудительное отображение деталей');
console.log('- testRenderQuizDetails() - Тестовая отрисовка деталей');
console.log('- checkQuizCardHandlers() - Проверка обработчиков карточек');
console.log('- checkHtmlStructure() - Проверка структуры HTML'); 