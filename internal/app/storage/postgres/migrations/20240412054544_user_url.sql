-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS users(
    id uuid NOT NULL UNIQUE,
    login VARCHAR(50) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    PRIMARY KEY (id, login)
);
CREATE INDEX IF NOT EXISTS idx_user_login ON users(login);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX idx_user_login;
DROP TABLE IF EXISTS users;
-- +goose StatementEnd