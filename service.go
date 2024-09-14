package bankxgo

import (
	"io"

	"github.com/bwmarrin/snowflake"
	"github.com/shopspring/decimal"
)

type ChargeReq struct {
	Amount decimal.Decimal `json:"amount"`
	AcctID snowflake.ID
	Email  string
}

type BalanceReq struct {
	AcctID snowflake.ID
	Email  string
}

type StatementReq struct {
	AcctID snowflake.ID
	Email  string
}

type Service interface {
	Deposit(ChargeReq) error
	Withdraw(ChargeReq) error
	Balance(BalanceReq) (decimal.Decimal, error)
	Statement(io.Writer, StatementReq) error
}

func NewService() *serviceImpl {
	return &serviceImpl{}
}

type serviceImpl struct{}

func (s *serviceImpl) Deposit(req ChargeReq) error {
	return nil
}

func (s *serviceImpl) Withdraw(req ChargeReq) error {
	return nil
}

func (s *serviceImpl) Balance(req BalanceReq) (decimal.Decimal, error) {
	return decimal.NewFromInt(0), nil
}

func (s *serviceImpl) Statement(w io.Writer, req StatementReq) error {
	return nil
}
