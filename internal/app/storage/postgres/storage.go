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
	usecase "github.com/avGenie/go-loyalty-system/internal/app/usecase/order"
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
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create transaction in postgres while creating user: %w", err)
	}
	defer tx.Rollback()

	queryInsertUser := `INSERT INTO users VALUES(@userID, @login, @password)`
	args := pgx.NamedArgs{
		"userID":   user.ID.String(),
		"login":    user.Login,
		"password": user.Password,
	}

	err = s.execInsertContext(ctx, err_api.ErrLoginExists, queryInsertUser, args)
	if err != nil {
		return fmt.Errorf("error while inserting user: %w", err)
	}

	queryInsertUserBalance := `INSERT INTO balance(user_id) VALUES($1)`
	err = s.execInsertContext(ctx, err_api.ErrUserExistsTable, queryInsertUserBalance, user.ID)
	if err != nil {
		return fmt.Errorf("error while inserting user balance: %w", err)
	}

	queryInsertUserWithdrawnBalance := `INSERT INTO withdrawn_balance(user_id) VALUES($1)`
	err = s.execInsertContext(ctx, err_api.ErrUserExistsTable, queryInsertUserWithdrawnBalance, user.ID)
	if err != nil {
		return fmt.Errorf("error while inserting user balance withdrawans: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction in postgres while creating user: %w", err)
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

	if len(orders) == 0 {
		return nil, err_api.ErrOrderForUserNotFound
	}

	return orders, nil
}

func (s *Postgres) UpdateOrders(ctx context.Context, orders entity.UpdateUserOrders) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction while updating orders in postgres: %w", err)
	}
	defer tx.Rollback()

	selectQuery := `SELECT status, accrual FROM orders WHERE number=$1 FOR UPDATE`
	stmtSelect, err := tx.PrepareContext(ctx, selectQuery)
	if err != nil {
		return fmt.Errorf("failed to prepare select query while updating orders in postgres: %w", err)
	}

	updateOrderQuery := `UPDATE orders SET status=$1::order_status, accrual=$2 WHERE number=$3`
	stmtUpdateOrder, err := tx.PrepareContext(ctx, updateOrderQuery)
	if err != nil {
		return fmt.Errorf("failed to prepare update query while updating orders in postgres: %w", err)
	}

	queryUpdateBalance := `UPDATE balance SET sum=sum+$1 WHERE user_id=$2`
	stmtUpdateBalance, err := tx.PrepareContext(ctx, queryUpdateBalance)
	if err != nil {
		return fmt.Errorf("failed to prepare update query while updating user balances in postgres: %w", err)
	}

	for _, order := range orders {
		row := stmtSelect.QueryRowContext(ctx, order.Order.Number)
		if row == nil {
			return fmt.Errorf("error while postgres request preparation while selecting order for update")
		}
	
		if row.Err() != nil {
			return fmt.Errorf("error while postgres request execution while selecting order for update: %w", row.Err())
		}
	
		var status string
		var accrual float64
		err := row.Scan(&status, &accrual)
		if err != nil {
			if err == sql.ErrNoRows {
				return err_api.ErrOrderNumberNotFound
			}
			return fmt.Errorf("error while processing response row in postgres while selecting order for update: %w", err)
		}

		if !usecase.IsUpdatableAccrualStatus(order.Order.Status, entity.OrderStatus(status)) {
			continue
		}

		_, err = stmtUpdateOrder.ExecContext(ctx, order.Order.Status, order.Order.Accrual, order.Order.Number)
		if err != nil {
			return fmt.Errorf("failed to update query while updating orders in postgres: %w", err)
		}

		if accrual == 0 && entity.StatusProcessedOrder == order.Order.Status {
			_, err = stmtUpdateBalance.ExecContext(ctx, order.Order.Accrual, order.UserID)
			if err != nil {
				return fmt.Errorf("failed to update query while updating orders in postgres: %w", err)
			}
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("unable to commit transaction while updating orders in postgres: %w", err)
	}

	return nil
}

func (s *Postgres) GetUserBalance(ctx context.Context, userID entity.UserID) (entity.UserBalance, error) {
	query := `SELECT b.sum, wb.withdrawn FROM balance AS b
				INNER JOIN withdrawn_balance AS wb
					ON b.user_id=wb.user_id 
			  WHERE b.user_id=$1`

	row := s.db.QueryRowContext(ctx, query, userID)
	if row == nil {
		return entity.UserBalance{}, fmt.Errorf("error while postgres request preparation while getting user balance")
	}

	if row.Err() != nil {
		return entity.UserBalance{}, fmt.Errorf("error while postgres request execution while getting user balance: %w", row.Err())
	}

	userBalance := entity.UserBalance{
		UserID: userID,
	}
	err := row.Scan(&userBalance.Balance, &userBalance.Withdrawans)
	if err != nil {
		if err == sql.ErrNoRows {
			return entity.UserBalance{}, err_api.ErrUserNotFoundTable
		}
		return entity.UserBalance{}, fmt.Errorf("error while processing response row in postgres while getting user balance: %w", err)
	}

	return userBalance, nil
}

func (s *Postgres) WithdrawUser(ctx context.Context, userID entity.UserID, withdraw entity.Withdraw) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction while withdrawing user bonuses in postgres: %w", err)
	}
	defer tx.Rollback()

	sum, err := s.selectUserBalanceOnUpdate(ctx, userID)
	if err != nil {
		return fmt.Errorf("error while withdrawing user bonuses in postgres: %w", err)
	}

	diffSum := sum - withdraw.Sum
	if diffSum < 0 {
		return err_api.ErrNotEnoughSum
	}

	queryInsertWithdrawal := `INSERT INTO withdrawals(order_number, sum) VALUES(@orderNumber, @sum)`
	argsWithdrawal := pgx.NamedArgs{
		"orderNumber": withdraw.OrderNumber,
		"sum":         withdraw.Sum,
	}
	err = s.execInsertContext(ctx, err_api.ErrOrderNumberExists, queryInsertWithdrawal, argsWithdrawal)
	if err != nil {
		return fmt.Errorf("error while inserting user withdrawal while withdrawing user bonuses in postgres: %w", err)
	}

	queryInsertUserWithdrawals := `INSERT INTO users_withdrawals VALUES($1, $2)`
	err = s.execInsertContext(ctx, err_api.ErrOrderNumberExists, queryInsertUserWithdrawals, userID, withdraw.OrderNumber)
	if err != nil {
		return fmt.Errorf("error while inserting user withdrawals while withdrawing user bonuses in postgres: %w", err)
	}

	queryUpdateBalance := `UPDATE balance SET sum=$1 WHERE user_id=$2`
	_, err = s.db.ExecContext(ctx, queryUpdateBalance, diffSum, userID)
	if err != nil {
		return fmt.Errorf("error while updating balance while withdrawing user bonuses in postgres: %w", err)
	}

	queryUpdateWithdrawBalance := `UPDATE withdrawn_balance SET withdrawn=withdrawn+$1 WHERE user_id=$2`
	_, err = s.db.ExecContext(ctx, queryUpdateWithdrawBalance, withdraw.Sum, userID)
	if err != nil {
		return fmt.Errorf("error while updating withdrawn sum while withdrawing user bonuses in postgres: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("unable to commit transaction while withdrawing user bonuses in postgres: %w", err)
	}

	return nil
}

func (s *Postgres) GetUserWithdrawals(ctx context.Context, userID entity.UserID) (entity.Withdrawals, error) {
	query := `SELECT w.order_number, w.sum, w.process_date FROM users_withdrawals AS uw
				JOIN withdrawals AS w
					ON uw.order_number=w.order_number
			  WHERE uw.user_id=$1
			  ORDER BY w.process_date`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("error in postgres request execution while getting user withdrawals: %w", err)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("error in postgres requested rows while getting user withdrawals: %w", rows.Err())
	}

	var withdrawals entity.Withdrawals
	for rows.Next() {
		var withdraw entity.Withdraw
		err := rows.Scan(&withdraw.OrderNumber, &withdraw.Sum, &withdraw.DateCreated)
		if err != nil {
			return nil, fmt.Errorf("error while parsing row while getting users withdrawals from postgres: %w", err)
		}

		withdrawals = append(withdrawals, withdraw)
	}

	if len(withdrawals) == 0 {
		return nil, err_api.ErrWithdrawalsForUserNotFound
	}

	return withdrawals, nil
}

func (s *Postgres) selectUserBalanceOnUpdate(ctx context.Context, userID entity.UserID) (float64, error) {
	querySelect := `SELECT sum FROM balance WHERE user_id=$1 FOR UPDATE`
	row := s.db.QueryRowContext(ctx, querySelect, userID)
	if row == nil {
		return 0, fmt.Errorf("error while postgres request preparation while selecting user balance")
	}

	if row.Err() != nil {
		return 0, fmt.Errorf("error while postgres request execution while selecting user balance: %w", row.Err())
	}

	var sum float64
	err := row.Scan(&sum)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, err_api.ErrUserNotFoundTable
		}

		return 0, fmt.Errorf("error while processing response row in postgres: %w", err)
	}

	return sum, nil
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

func (s *Postgres) execInsertContext(ctx context.Context, constraintErr error, query string, args ...any) error {
	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) {
			return fmt.Errorf("constraint error while inserting row to postgres: %w", constraintErr)
		}

		return fmt.Errorf("unable to insert row to postgres: %w", err)
	}

	return nil
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
