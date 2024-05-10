-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS users_withdrawals(
	user_id uuid REFERENCES users(id),
	order_number VARCHAR(16) REFERENCES withdrawals(order_number)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE users_withdrawals;
-- +goose StatementEnd
