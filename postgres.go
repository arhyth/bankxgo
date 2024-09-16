package bankxgo

import (
	"context"

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
		INSERT INTO charges (t, amount, tx_id, acct_id)
		VALUES ('debit', $1, $2, $3);
	`

	pgCreditChargeSQL = `
		INSERT INTO charges (t, amount, tx_id, acct_id)
		VALUES ('credit', $1, $2, $3);
	`

	pgSelectForUpdateAcctSQL = `
		SELECT balance INTO acct_balance
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

func NewPostgresEndpoint(connStr string) (*PostgresEndpoint, error) {
	cfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		return nil, err
	}

	if err = pool.Ping(context.Background()); err != nil {
		return nil, err
	}

	endpt := &PostgresEndpoint{
		pool: pool,
	}
	return endpt, err
}

func (pg *PostgresEndpoint) CreditUser(amount decimal.Decimal, userAcct, sysAcct snowflake.ID) error {
	ctx := context.Background()
	conn, err := pg.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	tx, err := conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return err
	}

	row := tx.QueryRow(ctx, pgInsertTxnSQL, "deposit")
	var itxn int64
	if err = row.Scan(&itxn); err != nil {
		return err
	}

	batch := &pgx.Batch{}
	batch.Queue(pgDebitChargeSQL, amount, itxn, sysAcct)
	batch.Queue(pgCreditChargeSQL, amount, itxn, userAcct)
	btresults := tx.SendBatch(ctx, batch)
	for i := 0; i < 2; i++ {
		if _, err = btresults.Exec(); err != nil {
			if err = tx.Rollback(ctx); err != nil {
				pg.log.Err(err).Msgf("transaction `%v` rollback fail", itxn)
			}
			return err
		}
	}
	defer btresults.Close()

	row = tx.QueryRow(ctx, pgSelectForUpdateAcctSQL, userAcct)
	var bal decimal.Decimal
	if err = row.Scan(&bal); err != nil {
		return err
	}

	if bal.LessThan(amount) {
		if err = tx.Rollback(ctx); err != nil {
			pg.log.Err(err).Msgf("transaction `%v` rollback fail", itxn)
		}
		return ErrBadRequest{Fields: map[string]string{"amount": "insufficient balance"}}
	}

	if _, err = tx.Exec(ctx, pgUpdateAcctSQL, bal.Add(amount.Neg()), userAcct); err != nil {
		if err = tx.Rollback(ctx); err != nil {
			pg.log.Err(err).Msgf("transaction `%v` rollback fail", itxn)
		}
		return err
	}

	return err
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

func (pg *PostgresEndpoint) GetAcct(id snowflake.ID) (*Account, error) {
	ctx := context.Background()
	conn, err := pg.pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	sql := `
	SELECT currency, balance
	FROM accounts
	WHERE pub_id = $1;
	`

	row := conn.QueryRow(ctx, sql, id)
	var (
		rcur string
		rbal decimal.Decimal
	)
	if err = row.Scan(&rcur, &rbal); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound{}
		}
		return nil, err
	}

	acct := &Account{
		acctID:   id,
		currency: rcur,
		balance:  rbal,
	}
	return acct, err
}
