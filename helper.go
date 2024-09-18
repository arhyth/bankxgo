package bankxgo

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/bwmarrin/snowflake"
	"github.com/jackc/pgx/v5"
)

type LocalHelper struct {
	Conn     *pgx.Conn
	SysAccts map[string]snowflake.ID
}

func NewLocalHelper(cfg *Config) (*LocalHelper, error) {
	conn, err := pgx.Connect(context.Background(), cfg.Database.ConnStr)
	if err != nil {
		return nil, err
	}

	sysAcctSS := make(map[string]snowflake.ID, len(cfg.SystemAccounts))
	for k, v := range cfg.SystemAccounts {
		id, err := snowflake.ParseString(v)
		if err != nil {
			return nil, err
		}
		sysAcctSS[strings.ToUpper(k)] = id
	}
	return &LocalHelper{
		Conn:     conn,
		SysAccts: sysAcctSS,
	}, nil
}

func (lh *LocalHelper) InitDB() (func(), error) {
	initSQLpath := filepath.Join("testdata", "init_db.sql")
	bits, err := os.ReadFile(initSQLpath)
	if err != nil {
		return nil, err
	}
	if _, err = lh.Conn.Exec(context.Background(), string(bits)); err != nil {
		return nil, err
	}
	return lh.teardownDB(), err
}

func (lh *LocalHelper) PrepareSystemAccounts() error {
	funcMap := template.FuncMap{
		"ToLower": strings.ToLower,
		"add":     func(a, b int) int { return a + b },
	}
	seedPath := filepath.Join("testdata", "seed_system_accounts.tmpl")
	bits, err := os.ReadFile(seedPath)
	if err != nil {
		return err
	}
	tmpl, err := template.New("seed_system_accounts").Funcs(funcMap).Parse(string(bits))
	if err != nil {
		return err
	}
	buf := new(bytes.Buffer)
	if err = tmpl.Execute(buf, lh.SysAccts); err != nil {
		return err
	}

	if _, err = lh.Conn.Exec(context.Background(), buf.String()); err != nil {
		return err
	}

	return err
}

func (lh *LocalHelper) teardownDB() func() {
	return func() {
		defer lh.Conn.Close(context.Background())

		tearSQLpath := filepath.Join("testdata", "teardown_db.sql")
		bits, err := os.ReadFile(tearSQLpath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "DB cleanup read teardown sql: %s", err.Error())
			return
		}
		if _, err = lh.Conn.Exec(context.Background(), string(bits)); err != nil {
			fmt.Fprintf(os.Stderr, "DB cleanup exec teardown sql: %s", err.Error())
			return
		}
	}
}
