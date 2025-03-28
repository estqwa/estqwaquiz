-- Таблица пользователей
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(100) NOT NULL UNIQUE,
    password VARCHAR(100) NOT NULL,
    profile_picture VARCHAR(255) DEFAULT '',
    games_played INT DEFAULT 0,
    total_score INT DEFAULT 0,
    highest_score INT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Таблица викторин
CREATE TABLE IF NOT EXISTS quizzes (
    id SERIAL PRIMARY KEY,
    title VARCHAR(100) NOT NULL,
    description VARCHAR(500) DEFAULT '',
    scheduled_time TIMESTAMP WITH TIME ZONE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'scheduled',
    question_count INT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Таблица вопросов
CREATE TABLE IF NOT EXISTS questions (
    id SERIAL PRIMARY KEY,
    quiz_id INT NOT NULL,
    text VARCHAR(500) NOT NULL,
    options JSONB NOT NULL,
    correct_option INT NOT NULL,
    time_limit_sec INT NOT NULL DEFAULT 10,
    point_value INT NOT NULL DEFAULT 10,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    FOREIGN KEY (quiz_id) REFERENCES quizzes (id) ON DELETE CASCADE
);

-- Таблица ответов пользователей
CREATE TABLE IF NOT EXISTS user_answers (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL,
    quiz_id INT NOT NULL,
    question_id INT NOT NULL,
    selected_option INT NOT NULL,
    is_correct BOOLEAN NOT NULL,
    response_time_ms BIGINT NOT NULL,
    score INT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    FOREIGN KEY (quiz_id) REFERENCES quizzes (id) ON DELETE CASCADE,
    FOREIGN KEY (question_id) REFERENCES questions (id) ON DELETE CASCADE
);

-- Таблица результатов
CREATE TABLE IF NOT EXISTS results (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL,
    quiz_id INT NOT NULL,
    username VARCHAR(50) NOT NULL,
    profile_picture VARCHAR(255) DEFAULT '',
    score INT NOT NULL,
    correct_answers INT NOT NULL,
    total_questions INT NOT NULL,
    rank INT DEFAULT 0,
    completed_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    FOREIGN KEY (quiz_id) REFERENCES quizzes (id) ON DELETE CASCADE,
    UNIQUE (user_id, quiz_id)
);

-- Индексы для оптимизации запросов
CREATE INDEX idx_user_answers_user_id ON user_answers (user_id);
CREATE INDEX idx_user_answers_quiz_id ON user_answers (quiz_id);
CREATE INDEX idx_results_quiz_id ON results (quiz_id);
CREATE INDEX idx_results_user_id ON results (user_id);
CREATE INDEX idx_questions_quiz_id ON questions (quiz_id);
CREATE INDEX idx_quizzes_status ON quizzes (status);
CREATE INDEX idx_quizzes_scheduled_time ON quizzes (scheduled_time);

-- Таблица инвалидированных токенов
CREATE TABLE IF NOT EXISTS invalid_tokens (
    user_id INT NOT NULL,
    invalidation_time TIMESTAMP WITH TIME ZONE NOT NULL,
    PRIMARY KEY (user_id),
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);