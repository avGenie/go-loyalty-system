package storage

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/avGenie/go-loyalty-system/internal/app/entity"
	err_api "github.com/avGenie/go-loyalty-system/internal/app/storage/api/errors"
	"github.com/avGenie/go-loyalty-system/internal/app/storage/api/model"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"go.uber.org/zap"
)

const (
	migrationDB     = "postgres"
	migrationFolder = "migrations"
)

//go:embed migrations/*.sql
var migrationFs embed.FS

type Postgres struct {
	model.Storage

	db *sql.DB
}

func NewPostgresStorage(dbStorageConnect string) (*Postgres, error) {
	db, err := sql.Open("pgx", dbStorageConnect)
	if err != nil {
		return nil, fmt.Errorf("error while postgresql connect: %w", err)
	}

	err = migration(db)
	if err != nil {
		return nil, fmt.Errorf("error while postgresql migration: %w", err)
	}

	return &Postgres{
		db: db,
	}, nil
}

func (s *Postgres) Close() error {
	err := s.db.Close()
	if err != nil {
		zap.L().Error("error while closing postgres storage", zap.Error(err))

		return fmt.Errorf("couldn'r closed postgres db: %w", err)
	}

	return nil
}

func (s *Postgres) CreateUser(ctx context.Context, user entity.User) error {
	query := `INSERT INTO users VALUES(@userID, @login, @password)`
	args := pgx.NamedArgs{
		"userID":   user.ID.String(),
		"login":    user.Login,
		"password": user.Password,
	}

	_, err := s.db.ExecContext(ctx, query, args)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) {
			return fmt.Errorf("error while save url to postgres: %w", err_api.ErrLoginExists)
		}

		return fmt.Errorf("unable to insert row to postgres: %w", err)
	}

	return nil
}

func (s *Postgres) GetUser(ctx context.Context, user entity.User) (entity.User, error) {
	query := `SELECT id, password FROM users WHERE login=@login`
	args := pgx.NamedArgs{
		"login": user.Login,
	}

	row := s.db.QueryRowContext(ctx, query, args)
	if row == nil {
		return user, fmt.Errorf("error while postgres request preparation while getting user")
	}

	if row.Err() != nil {
		return user, fmt.Errorf("error while postgres request execution while getting user: %w", row.Err())
	}

	err := row.Scan(&user.ID, &user.Password)
	if err != nil {
		if err == sql.ErrNoRows {
			return user, err_api.ErrLoginNotFound
		}
		return user, fmt.Errorf("error while processing response row in postgres while getting user: %w", err)
	}

	return user, nil
}

func (s *Postgres) UploadOrder(ctx context.Context, userID entity.UserID, orderNumber entity.OrderNumber) (entity.UserID, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return entity.UserID(""), fmt.Errorf("failed to create transaction in postgres while uploading order: %w", err)
	}
	defer tx.Rollback()

	queryInsertOrder := `INSERT INTO orders(number) VALUES(@number)`
	args := pgx.NamedArgs{
		"number": orderNumber,
	}

	_, err = s.db.ExecContext(ctx, queryInsertOrder, args)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) {
			userID, err := s.getUserIDByOrderNumber(ctx, orderNumber)
			if err != nil {
				return entity.UserID(""), fmt.Errorf("error in postgres while uploading order: %w", err)
			}

			return userID, err_api.ErrOrderNumberExists
		}

		return entity.UserID(""), fmt.Errorf("unable to insert row to postgres while uploading order: %w", err)
	}

	queryInsertOrderUser := `INSERT INTO users_orders VALUES(@user_id, @order_number)`
	args = pgx.NamedArgs{
		"user_id":      userID,
		"order_number": orderNumber,
	}

	_, err = s.db.ExecContext(ctx, queryInsertOrderUser, args)
	if err != nil {
		return entity.UserID(""), fmt.Errorf("unable to insert user id and order number to users_orders table in postgres while uploading order: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return entity.UserID(""), fmt.Errorf("failed to commit transaction in postgres while uploading order: %w", err)
	}

	return entity.UserID(""), nil
}

func (s *Postgres) GetUserOrders(ctx context.Context, userID entity.UserID) (entity.Orders, error) {
	query := `SELECT o.number, o.status, o.accrual, o.date_created FROM orders AS o
				JOIN users_orders AS uo
					ON o.number=uo.order_number
			  WHERE uo.user_id=$1`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("error in postgres request execution while getting user orders: %w", err)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("error in postgres requested rows while getting user orders: %w", rows.Err())
	}

	var orders entity.Orders
	for rows.Next() {
		var order entity.Order
		err := rows.Scan(&order.Number, &order.Status, &order.Accrual, &order.DateCreated)
		if err != nil {
			return nil, fmt.Errorf("error while parsing row while getting users order from postgres: %w", err)
		}

		orders = append(orders, order)
	}

	return orders, nil
}

func (s *Postgres) UpdateOrders(ctx context.Context, orders entity.Orders) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction while updating orders in postgres: %w", err)
	}
	defer tx.Rollback()

	selectQuery := `SELECT * FROM orders WHERE number=$1`
	stmtSelect, err := tx.PrepareContext(ctx, selectQuery)
	if err != nil {
		return fmt.Errorf("failed to prepare select query while updating orders in postgres: %w", err)
	}

	updateQuery := `UPDATE orders SET status=$1::order_status, accrual=$2 WHERE number=$3`
	stmtUpdate, err := tx.PrepareContext(ctx, updateQuery)
	if err != nil {
		return fmt.Errorf("failed to prepare update query while updating orders in postgres: %w", err)
	}

	for _, order := range orders {
		_, err = stmtSelect.ExecContext(ctx, order.Number)
		if err != nil {
			return fmt.Errorf("failed to select query while updating orders in postgres: %w", err)
		}

		_, err = stmtUpdate.ExecContext(ctx, order.Status, order.Accrual, order.Number)
		if err != nil {
			return fmt.Errorf("failed to update query while updating orders in postgres: %w", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("unable to commit transaction while updating orders in postgres: %w", err)
	}

	return nil
}


func (s *Postgres) UpdateBalanceBatch(ctx context.Context, balances entity.UserBalances) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction while updating user balances in postgres: %w", err)
	}
	defer tx.Rollback()

	queryUpdate := `UPDATE balance SET sum=sum+$1 WHERE user_id=$2`
	stmtUpdate, err := tx.PrepareContext(ctx, queryUpdate)
	if err != nil {
		return fmt.Errorf("failed to prepare update query while updating user balances in postgres: %w", err)
	}

	queryInsert := `INSERT INTO balance VALUES($1, $2)`
	stmtInsert, err := tx.PrepareContext(ctx, queryInsert)
	if err != nil {
		return fmt.Errorf("failed to prepare insert query while updating user balances in postgres: %w", err)
	}

	for _, balance := range balances {
		err := s.selectUserBalance(ctx, balance)
		if err != nil {
			if err != sql.ErrNoRows {
				return fmt.Errorf("failed to execute select query while updating user balances in postgres: %w", err)
			}

			_, err = stmtInsert.ExecContext(ctx, balance.UserID, balance.Balance)
			if err != nil {
				return fmt.Errorf("failed to execute insert query while updating user balances in postgres: %w", err)
			}
			continue
		}

		_, err = stmtUpdate.ExecContext(ctx, balance.Balance, balance.UserID)
		if err != nil {
			return fmt.Errorf("failed to execute update query while updating user balances in postgres: %w", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("unable to commit transaction while updating user balances in postgres: %w", err)
	}

	return nil
}

func (s *Postgres) selectUserBalance(ctx context.Context, balance entity.UserBalance) error {
	querySelect := `SELECT user_id FROM balance WHERE user_id=$1 FOR UPDATE`
	row := s.db.QueryRowContext(ctx, querySelect, balance.UserID)
	if row == nil {
		return fmt.Errorf("error while postgres request preparation while selecting user balance")
	}

	if row.Err() != nil {
		return fmt.Errorf("error while postgres request execution while selecting user balance: %w", row.Err())
	}

	var storageUserID string
	return row.Scan(&storageUserID)
}

func (s *Postgres) getUserIDByOrderNumber(ctx context.Context, orderNumber entity.OrderNumber) (entity.UserID, error) {
	query := `SELECT uo.user_id FROM orders AS o
				JOIN users_orders AS uo
					ON o.number=uo.order_number
			  WHERE o.number=$1`
	row := s.db.QueryRowContext(ctx, query, orderNumber)
	if row == nil {
		return entity.UserID(""), fmt.Errorf("error while postgres request preparation while getting user id by order number")
	}

	if row.Err() != nil {
		return entity.UserID(""), fmt.Errorf("error while postgres request execution while getting user id by order number: %w", row.Err())
	}

	var userID entity.UserID
	err := row.Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return entity.UserID(""), err_api.ErrOrderNumberNotFound
		}

		return entity.UserID(""), fmt.Errorf("error while processing response row in postgres: %w", err)
	}

	return userID, nil
}

func migration(db *sql.DB) error {
	goose.SetBaseFS(migrationFs)

	if err := goose.SetDialect(migrationDB); err != nil {
		return err
	}

	if err := goose.Up(db, migrationFolder); err != nil {
		return err
	}

	return nil
}
