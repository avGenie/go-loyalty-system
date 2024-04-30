-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS users_withdrawals(
	user_id uuid REFERENCES users(id) ON DELETE CASCADE,
	order_number VARCHAR(16) REFERENCES withdrawals(order_number) ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE users_withdrawals;
-- +goose StatementEnd
