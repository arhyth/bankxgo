package bankxgo_test

import (
	"testing"

	"github.com/arhyth/bankxgo"
	"github.com/arhyth/bankxgo/mocks"
	"github.com/bwmarrin/snowflake"
	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNewService(t *testing.T) {
	t.Run("returns an error when a system account does not exist", func(tt *testing.T) {
		as := assert.New(tt)
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockRepository(ctrl)
		sysAccts := map[string]snowflake.ID{
			"USD": snowflake.ParseInt64(7241301734201495552),
		}
		log := zerolog.Nop()
		repo.EXPECT().
			GetAccount(sysAccts["USD"]).
			Return(nil, bankxgo.ErrNotFound{})
		_, err := bankxgo.NewService(repo, sysAccts, &log)
		as.NotNil(err)
	})
}

func TestBalance(t *testing.T) {
	t.Run("returns decimal.Decimal amount on success", func(tt *testing.T) {
		as := assert.New(tt)
		reqrd := require.New(tt)
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockRepository(ctrl)
		sysAccts := map[string]snowflake.ID{
			"USD": snowflake.ParseInt64(7241301734201495552),
		}
		usdAcct := &bankxgo.Account{
			AcctID:   sysAccts["USD"],
			Currency: "USD",
		}
		repo.EXPECT().
			GetAccount(sysAccts["USD"]).
			Return(usdAcct, nil)
		userDeposit := decimal.New(1234, 0)
		userAcctID := snowflake.ParseInt64(7241407009730334720)
		userAcctCurr := "USD"
		repo.EXPECT().
			CreditUser(userDeposit, userAcctID, sysAccts["USD"]).
			Return(&userDeposit, nil)
		log := zerolog.Nop()
		svc, err := bankxgo.NewService(repo, sysAccts, &log)
		reqrd.Nil(err)

		repo.EXPECT().
			CreateAccount(gomock.AssignableToTypeOf(bankxgo.CreateAccountReq{})).
			Return(nil)
		userEmail := "newuser@balance.com"
		acr := bankxgo.CreateAccountReq{
			Email:    userEmail,
			AcctID:   userAcctID,
			Currency: userAcctCurr,
		}
		_, err = svc.CreateAccount(acr)
		reqrd.Nil(err)
		dep := bankxgo.ChargeReq{
			Amount:   userDeposit,
			AcctID:   userAcctID,
			Email:    userEmail,
			Currency: userAcctCurr,
		}
		bal, err := svc.Deposit(dep)
		reqrd.Nil(err)
		as.Equal(userDeposit, *bal)
	})
}
