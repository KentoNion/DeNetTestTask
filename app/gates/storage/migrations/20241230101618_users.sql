-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    nickname VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    score INT DEFAULT 0,
    registered TIMESTAMP DEFAULT NOW(),
    invited_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT users_email_key UNIQUE (email),
    CONSTRAINT users_nickname_key UNIQUE (nickname)
);

-- Создаём уникальный индекс для обеих колонок
CREATE UNIQUE INDEX unique_nickname_email ON users (nickname, email);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS users;
-- +goose StatementEnd