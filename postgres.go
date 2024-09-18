package bankxgo

import (
	"context"
	"fmt"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"
)

var (
	pgInsertTxnSQL = `
		INSERT INTO transactions (id, typ)
		VALUES (DEFAULT, $1)
		RETURNING id;
	`

	pgDebitChargeSQL = `
		INSERT INTO charges (typ, amount, tx_id, acct_id)
		VALUES ('debit', $1, $2, $3);
	`

	pgCreditChargeSQL = `
		INSERT INTO charges (typ, amount, tx_id, acct_id)
		VALUES ('credit', $1, $2, $3);
	`

	pgSelectForUpdateAcctSQL = `
		SELECT balance
		FROM accounts
		WHERE pub_id = $1
		FOR UPDATE;
	`

	pgUpdateAcctSQL = `
		UPDATE accounts
		SET balance = $1
		WHERE pub_id = $2;
	`
)

type PostgresEndpoint struct {
	pool *pgxpool.Pool
	log  *zerolog.Logger
}

var (
	_ Repository = (*PostgresEndpoint)(nil)
)

func NewPostgresEndpoint(connStr string, log *zerolog.Logger) (*PostgresEndpoint, error) {
	cfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 10
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		return nil, err
	}

	if err = pool.Ping(context.Background()); err != nil {
		return nil, err
	}

	endpt := &PostgresEndpoint{
		pool: pool,
		log:  log,
	}
	return endpt, err
}

func (pg *PostgresEndpoint) CreditUser(
	amount decimal.Decimal,
	userAcct,
	sysAcct snowflake.ID,
) (*decimal.Decimal, error) {
	// smoke test in case the service validation middleware
	// somehow is not wired up correctly
	if sysAcct == 0 {
		return nil, ErrInternalServer
	}

	ctx := context.Background()
	conn, err := pg.pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	tx, err := conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return nil, err
	}

	row := tx.QueryRow(ctx, pgInsertTxnSQL, "withdrawal")
	var itxn int64
	if err = row.Scan(&itxn); err != nil {
		return nil, err
	}

	if _, err = tx.Exec(ctx, pgDebitChargeSQL, amount, itxn, sysAcct); err != nil {
		if rerr := tx.Rollback(ctx); rerr != nil {
			pg.log.
				Err(rerr).
				Str("sql", "pgDebitChargeSQL").
				Msgf("transaction `%v` rollback fail", itxn)
		}
		return nil, fmt.Errorf("pgDebitChargeSQL: %w", err)
	}

	if _, err = tx.Exec(ctx, pgCreditChargeSQL, amount, itxn, userAcct); err != nil {
		if rerr := tx.Rollback(ctx); rerr != nil {
			pg.log.
				Err(rerr).
				Str("sql", "pgCreditChargeSQL").
				Msgf("transaction `%v` rollback fail", itxn)
		}
		return nil, fmt.Errorf("pgCreditChargeSQL: %w", err)
	}

	row = tx.QueryRow(ctx, pgSelectForUpdateAcctSQL, userAcct)
	var bal decimal.Decimal
	if err = row.Scan(&bal); err != nil {
		return nil, err
	}

	if bal.LessThan(amount) {
		if err = tx.Rollback(ctx); err != nil {
			pg.log.Err(err).Msgf("transaction `%v` rollback fail", itxn)
		}
		return nil, ErrBadRequest{Fields: map[string]string{"amount": "insufficient balance"}}
	}

	newbal := bal.Sub(amount)
	if _, err = tx.Exec(ctx, pgUpdateAcctSQL, newbal, userAcct); err != nil {
		if rerr := tx.Rollback(ctx); rerr != nil {
			pg.log.Err(rerr).Msgf("transaction `%v` rollback fail", itxn)
		}
		return nil, err
	}

	if err = tx.Commit(ctx); err != nil {
		pg.log.Err(err).Msg("CreditUser: transaction commit fail")
	}

	return &newbal, err
}

func (pg *PostgresEndpoint) DebitUser(
	amount decimal.Decimal,
	userAcct,
	sysAcct snowflake.ID,
) (*decimal.Decimal, error) {
	// smoke test in case the service validation middleware
	// somehow is not wired up correctly
	if sysAcct == 0 {
		return nil, ErrInternalServer
	}

	ctx := context.Background()
	conn, err := pg.pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	tx, err := conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return nil, err
	}

	row := tx.QueryRow(ctx, pgInsertTxnSQL, "deposit")
	var itxn int64
	if err = row.Scan(&itxn); err != nil {
		return nil, err
	}

	if _, err = tx.Exec(ctx, pgDebitChargeSQL, amount, itxn, userAcct); err != nil {
		if rerr := tx.Rollback(ctx); rerr != nil {
			pg.log.
				Err(rerr).
				Str("sql", "pgDebitChargeSQL").
				Msgf("transaction `%v` rollback fail", itxn)
		}
		return nil, fmt.Errorf("pgDebitChargeSQL: %w", err)
	}

	if _, err = tx.Exec(ctx, pgCreditChargeSQL, amount, itxn, sysAcct); err != nil {
		if rerr := tx.Rollback(ctx); rerr != nil {
			pg.log.
				Err(rerr).
				Str("sql", "pgCreditChargeSQL").
				Msgf("transaction `%v` rollback fail", itxn)
		}
		return nil, fmt.Errorf("pgCreditChargeSQL: %w", err)
	}

	row = tx.QueryRow(ctx, pgSelectForUpdateAcctSQL, userAcct)
	var bal decimal.Decimal
	if err = row.Scan(&bal); err != nil {
		return nil, err
	}

	newbal := bal.Add(amount)
	if _, err = tx.Exec(ctx, pgUpdateAcctSQL, newbal, userAcct); err != nil {
		if rerr := tx.Rollback(ctx); rerr != nil {
			pg.log.Err(rerr).Msgf("transaction `%v` rollback fail", itxn)
		}
		return nil, err
	}

	if err = tx.Commit(ctx); err != nil {
		pg.log.Err(err).Msg("DebitUser: transaction commit fail")
	}

	return &newbal, err
}

func (pg *PostgresEndpoint) CreateAccount(req CreateAccountReq) error {
	ctx := context.Background()
	conn, err := pg.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	sql := `
	INSERT INTO accounts (pub_id, email, currency)
	VALUES ($1, $2, $3);
	`

	if _, err = conn.Exec(ctx, sql, req.AcctID, req.Email, req.Currency); err != nil {
		return err
	}

	return err
}

func (pg *PostgresEndpoint) GetAccount(id snowflake.ID) (*Account, error) {
	ctx := context.Background()
	conn, err := pg.pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	sql := `
	SELECT email, currency, balance
	FROM accounts
	WHERE pub_id = $1;
	`

	row := conn.QueryRow(ctx, sql, id)
	var (
		rcur, remail string
		rbal         decimal.Decimal
	)
	if err = row.Scan(&remail, &rcur, &rbal); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound{ID: id.Int64()}
		}
		return nil, err
	}

	acct := &Account{
		AcctID:   id,
		Currency: rcur,
		Balance:  rbal,
		Email:    remail,
	}
	return acct, err
}

func (pg *PostgresEndpoint) GetAccountCharges(id snowflake.ID) ([]Charge, error) {
	ctx := context.Background()
	conn, err := pg.pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	sql := `
	SELECT amount, typ, created_at FROM charges
	WHERE acct_id = $1;
	`
	rows, err := conn.Query(ctx, sql, id)
	if err != nil {
		return nil, err
	}
	var (
		amt       decimal.Decimal
		typ       string
		createdAt time.Time
		collected []Charge
	)
	for rows.Next() {
		rows.Scan(&amt, &typ, &createdAt)
		collected = append(collected, Charge{
			Amount:    amt,
			Typ:       typ,
			CreatedAt: createdAt,
		})
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("charges rows.Scan: %w", err)
	}

	return collected, err
}
