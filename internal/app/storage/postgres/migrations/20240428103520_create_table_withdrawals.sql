-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS withdrawals(
	order_number VARCHAR(16),
	sum numeric NOT NULL,
	process_date TIMESTAMP NOT NULL DEFAULT now()
);
ALTER TABLE balance
   ADD CONSTRAINT positive_sum check (sum >= 0);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE withdrawals;
-- +goose StatementEnd
