package bankxgo_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/bwmarrin/snowflake"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/arhyth/bankxgo"
)

var (
	testDBConnStr string
)

func init() {
	testDBConnStr = os.Getenv("TEST_DB_CONN_STR")
}

func TestPostgres(t *testing.T) {
	as := assert.New(t)
	reqrd := require.New(t)

	conn, teardown, err := initDB()
	reqrd.Nil(err)
	t.Cleanup(teardown)
	node, err := snowflake.NewNode(111)
	reqrd.Nil(err)
	tst := &tester{
		conn: conn,
		node: node,
	}
	accts, err := tst.prepareSystemAccounts(conn, "USD", "PHP", "EUR")
	as.Nil(err)

	endpt, err := bankxgo.NewPostgresEndpoint(testDBConnStr)
	reqrd.Nil(err)

	t.Run("Deposit", func(tt *testing.T) {
		car := bankxgo.CreateAccountReq{
			Email:    "arhyth@gmail.com",
			Currency: "USD",
			AcctID:   node.Generate(),
		}
		endpt.CreateAccount(car)
		reqrd.Nil(err)

		amount := decimal.New(123, -1)
		err = endpt.CreditUser(amount, car.AcctID, accts[car.Currency])
		as.Nil(err)
	})
}

func initDB() (*pgx.Conn, func(), error) {
	conn, err := pgx.Connect(context.Background(), testDBConnStr)
	if err != nil {
		return nil, nil, err
	}
	initSQLpath := filepath.Join("testdata", "init_db.sql")
	bits, err := os.ReadFile(initSQLpath)
	if err != nil {
		return conn, nil, err
	}
	if _, err = conn.Exec(context.Background(), string(bits)); err != nil {
		return conn, nil, err
	}
	return conn, teardownDB(conn), err
}

func teardownDB(conn *pgx.Conn) func() {
	return func() {
		defer conn.Close(context.Background())

		tearSQLpath := filepath.Join("testdata", "teardown_db.sql")
		bits, err := os.ReadFile(tearSQLpath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "DB cleanup read teardown sql: %s", err.Error())
			return
		}
		if _, err = conn.Exec(context.Background(), string(bits)); err != nil {
			fmt.Fprintf(os.Stderr, "DB cleanup exec teardown sql: %s", err.Error())
			return
		}
	}
}

type tester struct {
	conn *pgx.Conn
	node *snowflake.Node
}

func (t *tester) prepareSystemAccounts(conn *pgx.Conn, currencies ...string) (map[string]snowflake.ID, error) {
	funcMap := template.FuncMap{
		"ToLower": strings.ToLower,
		"add":     func(a, b int) int { return a + b },
	}
	seedPath := filepath.Join("testdata", "seed_system_accounts.tmpl")
	bits, err := os.ReadFile(seedPath)
	if err != nil {
		return nil, err
	}
	tmpl, err := template.New("seed_system_accounts").Funcs(funcMap).Parse(string(bits))
	if err != nil {
		return nil, err
	}
	accts := make(map[string]snowflake.ID, len(currencies))
	inputForTemplate := make(map[string]string, len(currencies))
	node, err := snowflake.NewNode(111)
	if err != nil {
		return nil, err
	}
	for _, c := range currencies {
		sid := node.Generate()
		accts[c] = sid
		inputForTemplate[c] = sid.String()
	}

	buf := new(bytes.Buffer)
	if err = tmpl.Execute(buf, inputForTemplate); err != nil {
		return nil, err
	}

	if _, err = conn.Exec(context.Background(), buf.String()); err != nil {
		return nil, err
	}

	return accts, err
}
