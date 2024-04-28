-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS balance(
	user_id uuid REFERENCES users(id) ON DELETE CASCADE,
	sum numeric DEFAULT 0
);
CREATE TABLE IF NOT EXISTS withdrawn_balance(
	user_id uuid REFERENCES users(id) ON DELETE CASCADE,
	withdrawn numeric DEFAULT 0
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE balance;
DROP TABLE withdrawn_balance;
-- +goose StatementEnd
