-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS balance(
	user_id uuid REFERENCES users(id) ON DELETE CASCADE,
	sum numeric NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE balance;
-- +goose StatementEnd
