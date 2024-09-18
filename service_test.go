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
		log := zerolog.Nop()
		svc, err := bankxgo.NewService(repo, sysAccts, &log)
		reqrd.Nil(err)

		userEmail := "newuser@balance.com"
		acr := bankxgo.CreateAccountReq{
			Email:    userEmail,
			AcctID:   userAcctID,
			Currency: userAcctCurr,
		}
		repo.EXPECT().
			CreateAccount(gomock.AssignableToTypeOf(bankxgo.CreateAccountReq{})).
			Return(nil)
		_, err = svc.CreateAccount(acr)
		reqrd.Nil(err)
		dep := bankxgo.ChargeReq{
			Amount:   userDeposit,
			AcctID:   userAcctID,
			Email:    userEmail,
			Currency: userAcctCurr,
		}
		repo.EXPECT().
			DebitUser(userDeposit, userAcctID, sysAccts["USD"]).
			Return(&userDeposit, nil)
		bal, err := svc.Deposit(dep)
		reqrd.Nil(err)
		as.Equal(userDeposit, *bal)
	})
}

func TestWithdraw(t *testing.T) {
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
		log := zerolog.Nop()
		svc, err := bankxgo.NewService(repo, sysAccts, &log)
		reqrd.Nil(err)

		userEmail := "newuser@balance.com"
		acr := bankxgo.CreateAccountReq{
			Email:    userEmail,
			AcctID:   userAcctID,
			Currency: userAcctCurr,
		}
		repo.EXPECT().
			CreateAccount(gomock.AssignableToTypeOf(bankxgo.CreateAccountReq{})).
			Return(nil)
		_, err = svc.CreateAccount(acr)
		reqrd.Nil(err)
		dep := bankxgo.ChargeReq{
			Amount:   userDeposit,
			AcctID:   userAcctID,
			Email:    userEmail,
			Currency: userAcctCurr,
		}
		repo.EXPECT().
			DebitUser(userDeposit, userAcctID, sysAccts["USD"]).
			Return(&userDeposit, nil)
		bal, err := svc.Deposit(dep)
		reqrd.Nil(err)
		reqrd.Equal(userDeposit, *bal)

		withdraw := bankxgo.ChargeReq{
			Amount:   userDeposit.Sub(decimal.New(100, 0)),
			AcctID:   userAcctID,
			Email:    userEmail,
			Currency: userAcctCurr,
		}
		repo.EXPECT().
			CreditUser(withdraw.Amount, userAcctID, sysAccts["USD"]).
			Return(&withdraw.Amount, nil)
		bal, err = svc.Withdraw(withdraw)
		reqrd.Nil(err)
		as.Equal(withdraw.Amount, *bal)
	})
}
