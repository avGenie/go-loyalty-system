-- +goose Up
-- +goose StatementBegin
CREATE TYPE order_status AS ENUM('NEW', 'INVALID', 'PROCESSING', 'PROCESSED');
CREATE TABLE IF NOT EXISTS orders(
	number VARCHAR(16) PRIMARY KEY,
	status order_status NOT NULL DEFAULT 'NEW',
	accrual numeric DEFAULT 0,
	date_created TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS users_orders(
	user_id uuid REFERENCES users(id),
	order_number VARCHAR(16) REFERENCES orders(number)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TYPE order_status;
DROP TABLE orders;
DROP TABLE users_orders;
-- +goose StatementEnd
