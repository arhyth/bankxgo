//go:build integration
// +build integration

package bankxgo_test

import (
	"os"
	"testing"

	"github.com/bwmarrin/snowflake"
	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/arhyth/bankxgo"
)

var (
	testCfg string
)

func init() {
	testCfg = os.Getenv("BANKXGO_TEST_CONFIG")
}

func TestPostgres(t *testing.T) {
	as := assert.New(t)
	reqrd := require.New(t)

	var cfg bankxgo.Config
	cfgfl, err := os.Open(testCfg)
	reqrd.Nil(err)
	err = yaml.NewDecoder(cfgfl).Decode(&cfg)
	reqrd.Nil(err)

	lh, err := bankxgo.NewLocalHelper(&cfg)
	reqrd.Nil(err)
	teardown, err := lh.InitDB()
	reqrd.Nil(err)
	t.Cleanup(teardown)
	node, err := snowflake.NewNode(111)
	reqrd.Nil(err)
	err = lh.PrepareSystemAccounts()
	reqrd.Nil(err)

	log := zerolog.Nop()
	endpt, err := bankxgo.NewPostgresEndpoint(cfg.Database.ConnStr, &log)
	reqrd.Nil(err)

	t.Run("CreditUser", func(tt *testing.T) {
		car := bankxgo.CreateAccountReq{
			Email:    "arhyth@gmail.com",
			Currency: "USD",
			AcctID:   node.Generate(),
		}
		endpt.CreateAccount(car)
		reqrd.Nil(err)

		amount := decimal.New(123, 0)
		cbal, err := endpt.CreditUser(amount, car.AcctID, lh.SysAccts[car.Currency])
		reqrd.Nil(err)
		retrieved, err := endpt.GetAccount(car.AcctID)
		reqrd.Nil(err)
		as.Equal(retrieved.Balance, *cbal)
		as.Equal(amount, retrieved.Balance)
	})

	t.Run("DebitUser returns error on insufficient balance", func(tt *testing.T) {
		car := bankxgo.CreateAccountReq{
			Email:    "poor@guy.com",
			Currency: "PHP",
			AcctID:   node.Generate(),
		}
		endpt.CreateAccount(car)
		reqrd.Nil(err)

		amount := decimal.New(5000, 0)
		bal, err := endpt.DebitUser(amount, car.AcctID, lh.SysAccts[car.Currency])
		reqrd.ErrorAs(err, &bankxgo.ErrBadRequest{})
		as.Nil(bal)
	})
}
