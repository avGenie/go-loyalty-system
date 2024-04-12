-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS user {
    id uuid PRIMARY KEY,
    login VARCHAR(50) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL
};
CREATE INDEX IF NOT EXISTS idx_user_login ON user(login);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX idx_user_login;
DROP TABLE IF EXISTS user;
-- +goose StatementEnd
