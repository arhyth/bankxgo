package bankxgo_test

import (
	"bytes"
	"testing"

	"github.com/bwmarrin/snowflake"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/arhyth/bankxgo"
	"github.com/arhyth/bankxgo/mocks"
)

func TestValidationMWCreateAccount(t *testing.T) {
	t.Run("returns an error on a non-supported currency", func(tt *testing.T) {
		as := assert.New(tt)
		ctrl := gomock.NewController(tt)
		repo := mocks.NewMockRepository(ctrl)
		svc := mocks.NewMockService(ctrl)
		usdSysAcct := snowflake.ParseInt64(7241720446024945664)
		sysAccts := map[string]snowflake.ID{"USD": usdSysAcct}
		v := bankxgo.NewValidationMiddleware(repo, sysAccts)(svc)

		userAcctID := snowflake.ParseInt64(7241722241547767808)
		userEmail := "nopass@jpy.com"
		dep := bankxgo.CreateAccountReq{
			AcctID:   userAcctID,
			Email:    userEmail,
			Currency: "JPY",
		}
		acct, err := v.CreateAccount(dep)
		as.NotNil(err)
		as.Nil(acct)
	})

	t.Run("Returns error on invalid email format", func(tt *testing.T) {
		as := assert.New(tt)
		ctrl := gomock.NewController(tt)
		repo := mocks.NewMockRepository(ctrl)
		svc := mocks.NewMockService(ctrl)
		v := bankxgo.NewValidationMiddleware(repo, nil)(svc)
		userEmail := "g!bberis#"
		req := bankxgo.CreateAccountReq{
			Email:    userEmail,
			Currency: "PHP",
		}
		acct, err := v.CreateAccount(req)
		as.NotNil(err)
		as.Nil(acct)
	})
}

func TestValidationMWWithdraw(t *testing.T) {
	t.Run("returns error on non-existent account", func(tt *testing.T) {
		as := assert.New(tt)
		ctrl := gomock.NewController(tt)
		repo := mocks.NewMockRepository(ctrl)
		svc := mocks.NewMockService(ctrl)
		usdSysAcct := snowflake.ParseInt64(7241720446024945664)
		sysAccts := map[string]snowflake.ID{"USD": usdSysAcct}
		v := bankxgo.NewValidationMiddleware(repo, sysAccts)(svc)
		userAcctID := snowflake.ParseInt64(7241722241547767808)
		userEmail := "noaccount@bank.com"
		repo.EXPECT().
			GetAccount(userAcctID).
			Return(nil, bankxgo.ErrNotFound{ID: userAcctID.Int64()})
		req := bankxgo.ChargeReq{
			Amount: decimal.NewFromInt(123),
			AcctID: userAcctID,
			Email:  userEmail,
		}
		bal, err := v.Withdraw(req)
		as.NotNil(err)
		as.ErrorAs(err, &bankxgo.ErrNotFound{})
		as.Nil(bal)
	})

	t.Run("returns error on system account", func(tt *testing.T) {
		as := assert.New(tt)
		ctrl := gomock.NewController(tt)
		repo := mocks.NewMockRepository(ctrl)
		svc := mocks.NewMockService(ctrl)
		usdSysAcct := snowflake.ParseInt64(7241720446024945664)
		sysAccts := map[string]snowflake.ID{"USD": usdSysAcct}
		v := bankxgo.NewValidationMiddleware(repo, sysAccts)(svc)
		req := bankxgo.ChargeReq{
			Amount: decimal.NewFromInt(123),
			AcctID: usdSysAcct,
			Email:  "attacker@maybe.com",
		}
		bal, err := v.Withdraw(req)
		as.NotNil(err)
		as.Nil(bal)
	})

	t.Run("returns error on mismatched email", func(tt *testing.T) {
		as := assert.New(tt)
		ctrl := gomock.NewController(tt)
		repo := mocks.NewMockRepository(ctrl)
		svc := mocks.NewMockService(ctrl)
		usdSysAcct := snowflake.ParseInt64(7241720446024945664)
		sysAccts := map[string]snowflake.ID{"USD": usdSysAcct}
		v := bankxgo.NewValidationMiddleware(repo, sysAccts)(svc)

		userAcctID := snowflake.ParseInt64(7241722241547767808)
		userEmail := "mismatched@email.com"
		repo.EXPECT().
			GetAccount(userAcctID).
			Return(&bankxgo.Account{
				AcctID: userAcctID,
				Email:  "correct@email.com",
			}, nil)
		req := bankxgo.ChargeReq{
			Amount: decimal.NewFromInt(123),
			AcctID: userAcctID,
			Email:  userEmail,
		}
		bal, err := v.Withdraw(req)
		as.NotNil(err)
		as.Nil(bal)
	})

	t.Run("returns error on empty email", func(tt *testing.T) {
		as := assert.New(tt)
		ctrl := gomock.NewController(tt)
		repo := mocks.NewMockRepository(ctrl)
		svc := mocks.NewMockService(ctrl)
		usdSysAcct := snowflake.ParseInt64(7241720446024945664)
		sysAccts := map[string]snowflake.ID{"USD": usdSysAcct}
		v := bankxgo.NewValidationMiddleware(repo, sysAccts)(svc)

		userAcctID := snowflake.ParseInt64(7241722241547767808)
		userEmail := ""
		req := bankxgo.ChargeReq{
			Amount: decimal.NewFromInt(123),
			AcctID: userAcctID,
			Email:  userEmail,
		}
		bal, err := v.Withdraw(req)
		as.NotNil(err)
		as.Nil(bal)
	})

	t.Run("Withdraw returns error on negative amount", func(tt *testing.T) {
		as := assert.New(tt)
		ctrl := gomock.NewController(tt)
		repo := mocks.NewMockRepository(ctrl)
		svc := mocks.NewMockService(ctrl)
		usdSysAcct := snowflake.ParseInt64(7241720446024945664)
		sysAccts := map[string]snowflake.ID{"USD": usdSysAcct}
		v := bankxgo.NewValidationMiddleware(repo, sysAccts)(svc)

		userAcctID := snowflake.ParseInt64(7241722241547767808)
		userEmail := "negative@amount.com"
		dep := bankxgo.ChargeReq{
			Amount: decimal.NewFromInt(-123),
			AcctID: userAcctID,
			Email:  userEmail,
		}
		bal, err := v.Withdraw(dep)
		as.NotNil(err)
		as.Nil(bal)
	})

	t.Run("returns error on insufficient balance", func(tt *testing.T) {
		as := assert.New(tt)
		ctrl := gomock.NewController(tt)
		repo := mocks.NewMockRepository(ctrl)
		svc := mocks.NewMockService(ctrl)
		usdSysAcct := snowflake.ParseInt64(7241720446024945664)
		sysAccts := map[string]snowflake.ID{"USD": usdSysAcct}
		v := bankxgo.NewValidationMiddleware(repo, sysAccts)(svc)

		userAcctID := snowflake.ParseInt64(7241722241547767808)
		userEmail := "tinimbangpero@kulang.com"
		repo.EXPECT().
			GetAccount(userAcctID).
			Return(&bankxgo.Account{
				AcctID:  userAcctID,
				Email:   "tinimbangpero@kulang.com",
				Balance: decimal.NewFromInt(100),
			}, nil)
		req := bankxgo.ChargeReq{
			Amount: decimal.NewFromInt(123),
			AcctID: userAcctID,
			Email:  userEmail,
		}
		bal, err := v.Withdraw(req)
		as.NotNil(err)
		as.Nil(bal)
	})
}

func TestValidationMWDeposit(t *testing.T) {
	t.Run("returns error on non-existent account", func(tt *testing.T) {
		as := assert.New(tt)
		ctrl := gomock.NewController(tt)
		repo := mocks.NewMockRepository(ctrl)
		svc := mocks.NewMockService(ctrl)
		usdSysAcct := snowflake.ParseInt64(7241720446024945664)
		sysAccts := map[string]snowflake.ID{"USD": usdSysAcct}
		v := bankxgo.NewValidationMiddleware(repo, sysAccts)(svc)
		userAcctID := snowflake.ParseInt64(7241722241547767808)
		userEmail := "noaccount@bank.com"
		repo.EXPECT().
			GetAccount(userAcctID).
			Return(nil, bankxgo.ErrNotFound{ID: userAcctID.Int64()})
		req := bankxgo.ChargeReq{
			Amount: decimal.NewFromInt(123),
			AcctID: userAcctID,
			Email:  userEmail,
		}
		bal, err := v.Deposit(req)
		as.NotNil(err)
		as.ErrorAs(err, &bankxgo.ErrNotFound{})
		as.Nil(bal)
	})

	t.Run("returns error on system account", func(tt *testing.T) {
		as := assert.New(tt)
		ctrl := gomock.NewController(tt)
		repo := mocks.NewMockRepository(ctrl)
		svc := mocks.NewMockService(ctrl)
		usdSysAcct := snowflake.ParseInt64(7241720446024945664)
		sysAccts := map[string]snowflake.ID{"USD": usdSysAcct}
		v := bankxgo.NewValidationMiddleware(repo, sysAccts)(svc)
		req := bankxgo.ChargeReq{
			Amount: decimal.NewFromInt(123),
			AcctID: usdSysAcct,
			Email:  "attacker@maybe.com",
		}
		bal, err := v.Deposit(req)
		as.NotNil(err)
		as.Nil(bal)
	})

	t.Run("returns error on mismatched email", func(tt *testing.T) {
		as := assert.New(tt)
		ctrl := gomock.NewController(tt)
		repo := mocks.NewMockRepository(ctrl)
		svc := mocks.NewMockService(ctrl)
		usdSysAcct := snowflake.ParseInt64(7241720446024945664)
		sysAccts := map[string]snowflake.ID{"USD": usdSysAcct}
		v := bankxgo.NewValidationMiddleware(repo, sysAccts)(svc)

		userAcctID := snowflake.ParseInt64(7241722241547767808)
		userEmail := "mismatched@email.com"
		repo.EXPECT().
			GetAccount(userAcctID).
			Return(&bankxgo.Account{
				AcctID: userAcctID,
				Email:  "correct@email.com",
			}, nil)
		req := bankxgo.ChargeReq{
			Amount: decimal.NewFromInt(123),
			AcctID: userAcctID,
			Email:  userEmail,
		}
		bal, err := v.Deposit(req)
		as.NotNil(err)
		as.Nil(bal)
	})

	t.Run("returns error on empty email", func(tt *testing.T) {
		as := assert.New(tt)
		ctrl := gomock.NewController(tt)
		repo := mocks.NewMockRepository(ctrl)
		svc := mocks.NewMockService(ctrl)
		usdSysAcct := snowflake.ParseInt64(7241720446024945664)
		sysAccts := map[string]snowflake.ID{"USD": usdSysAcct}
		v := bankxgo.NewValidationMiddleware(repo, sysAccts)(svc)

		userAcctID := snowflake.ParseInt64(7241722241547767808)
		userEmail := ""
		req := bankxgo.ChargeReq{
			Amount: decimal.NewFromInt(123),
			AcctID: userAcctID,
			Email:  userEmail,
		}
		bal, err := v.Deposit(req)
		as.NotNil(err)
		as.Nil(bal)
	})

	t.Run("Deposit returns error on negative amount", func(tt *testing.T) {
		as := assert.New(tt)
		ctrl := gomock.NewController(tt)
		repo := mocks.NewMockRepository(ctrl)
		svc := mocks.NewMockService(ctrl)
		usdSysAcct := snowflake.ParseInt64(7241720446024945664)
		sysAccts := map[string]snowflake.ID{"USD": usdSysAcct}
		v := bankxgo.NewValidationMiddleware(repo, sysAccts)(svc)

		userAcctID := snowflake.ParseInt64(7241722241547767808)
		userEmail := "negative@amount.com"
		dep := bankxgo.ChargeReq{
			Amount: decimal.NewFromInt(-123),
			AcctID: userAcctID,
			Email:  userEmail,
		}
		bal, err := v.Deposit(dep)
		as.NotNil(err)
		as.Nil(bal)
	})
}

func TestValidationMWBalance(t *testing.T) {
	t.Run("returns error on non-existent account", func(tt *testing.T) {
		as := assert.New(tt)
		ctrl := gomock.NewController(tt)
		repo := mocks.NewMockRepository(ctrl)
		svc := mocks.NewMockService(ctrl)
		usdSysAcct := snowflake.ParseInt64(7241720446024945664)
		sysAccts := map[string]snowflake.ID{"USD": usdSysAcct}
		v := bankxgo.NewValidationMiddleware(repo, sysAccts)(svc)
		userAcctID := snowflake.ParseInt64(7241722241547767808)
		userEmail := "noaccount@bank.com"
		repo.EXPECT().
			GetAccount(userAcctID).
			Return(nil, bankxgo.ErrNotFound{ID: userAcctID.Int64()})
		req := bankxgo.BalanceReq{
			AcctID: userAcctID,
			Email:  userEmail,
		}
		bal, err := v.Balance(req)
		as.NotNil(err)
		as.ErrorAs(err, &bankxgo.ErrNotFound{})
		as.Nil(bal)
	})

	t.Run("returns error on mismatched email", func(tt *testing.T) {
		as := assert.New(tt)
		ctrl := gomock.NewController(tt)
		repo := mocks.NewMockRepository(ctrl)
		svc := mocks.NewMockService(ctrl)
		usdSysAcct := snowflake.ParseInt64(7241720446024945664)
		sysAccts := map[string]snowflake.ID{"USD": usdSysAcct}
		v := bankxgo.NewValidationMiddleware(repo, sysAccts)(svc)

		userAcctID := snowflake.ParseInt64(7241722241547767808)
		userEmail := "mismatched@email.com"
		repo.EXPECT().
			GetAccount(userAcctID).
			Return(&bankxgo.Account{
				AcctID: userAcctID,
				Email:  "correct@email.com",
			}, nil)
		req := bankxgo.BalanceReq{
			AcctID: userAcctID,
			Email:  userEmail,
		}
		bal, err := v.Balance(req)
		as.NotNil(err)
		as.Nil(bal)
	})

	t.Run("returns error on empty email", func(tt *testing.T) {
		as := assert.New(tt)
		ctrl := gomock.NewController(tt)
		repo := mocks.NewMockRepository(ctrl)
		svc := mocks.NewMockService(ctrl)
		usdSysAcct := snowflake.ParseInt64(7241720446024945664)
		sysAccts := map[string]snowflake.ID{"USD": usdSysAcct}
		v := bankxgo.NewValidationMiddleware(repo, sysAccts)(svc)

		userAcctID := snowflake.ParseInt64(7241722241547767808)
		userEmail := ""
		req := bankxgo.BalanceReq{
			AcctID: userAcctID,
			Email:  userEmail,
		}
		bal, err := v.Balance(req)
		as.NotNil(err)
		as.Nil(bal)
	})
}

func TestValidationMWStatement(t *testing.T) {
	t.Run("returns error on non-existent account", func(tt *testing.T) {
		as := assert.New(tt)
		ctrl := gomock.NewController(tt)
		repo := mocks.NewMockRepository(ctrl)
		svc := mocks.NewMockService(ctrl)
		usdSysAcct := snowflake.ParseInt64(7241720446024945664)
		sysAccts := map[string]snowflake.ID{"USD": usdSysAcct}
		v := bankxgo.NewValidationMiddleware(repo, sysAccts)(svc)
		userAcctID := snowflake.ParseInt64(7241722241547767808)
		userEmail := "noaccount@bank.com"
		repo.EXPECT().
			GetAccount(userAcctID).
			Return(nil, bankxgo.ErrNotFound{ID: userAcctID.Int64()})
		req := bankxgo.StatementReq{
			AcctID: userAcctID,
			Email:  userEmail,
		}
		w := &bytes.Buffer{}
		err := v.Statement(w, req)
		as.NotNil(err)
		as.ErrorAs(err, &bankxgo.ErrNotFound{})
	})

	t.Run("returns error on mismatched email", func(tt *testing.T) {
		as := assert.New(tt)
		ctrl := gomock.NewController(tt)
		repo := mocks.NewMockRepository(ctrl)
		svc := mocks.NewMockService(ctrl)
		usdSysAcct := snowflake.ParseInt64(7241720446024945664)
		sysAccts := map[string]snowflake.ID{"USD": usdSysAcct}
		v := bankxgo.NewValidationMiddleware(repo, sysAccts)(svc)

		userAcctID := snowflake.ParseInt64(7241722241547767808)
		userEmail := "mismatched@email.com"
		repo.EXPECT().
			GetAccount(userAcctID).
			Return(&bankxgo.Account{
				AcctID: userAcctID,
				Email:  "correct@email.com",
			}, nil)
		req := bankxgo.StatementReq{
			AcctID: userAcctID,
			Email:  userEmail,
		}
		w := &bytes.Buffer{}
		err := v.Statement(w, req)
		as.NotNil(err)
	})

	t.Run("returns error on empty email", func(tt *testing.T) {
		as := assert.New(tt)
		ctrl := gomock.NewController(tt)
		repo := mocks.NewMockRepository(ctrl)
		svc := mocks.NewMockService(ctrl)
		usdSysAcct := snowflake.ParseInt64(7241720446024945664)
		sysAccts := map[string]snowflake.ID{"USD": usdSysAcct}
		v := bankxgo.NewValidationMiddleware(repo, sysAccts)(svc)

		userAcctID := snowflake.ParseInt64(7241722241547767808)
		userEmail := ""
		req := bankxgo.StatementReq{
			AcctID: userAcctID,
			Email:  userEmail,
		}
		w := &bytes.Buffer{}
		err := v.Statement(w, req)
		as.NotNil(err)
	})
}
