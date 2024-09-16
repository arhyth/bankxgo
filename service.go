package bankxgo

import (
	"fmt"
	"io"

	"github.com/bwmarrin/snowflake"
	"github.com/shopspring/decimal"
)

type Account struct {
	// acctID and other instances of it in struct fields in this package
	// refers to the public id of the account (as opposed to its BIGINT id)
	// hence named `pub_id` column in the database
	acctID   snowflake.ID
	currency string
	balance  decimal.Decimal
}

type CreateAccountReq struct {
	Email    string `json:"email"`
	Currency string `json:"currency"`
	AcctID   snowflake.ID
}

type ChargeReq struct {
	Amount decimal.Decimal `json:"amount"`
	AcctID snowflake.ID
	Email  string

	// not passed from input but from middleware
	Currency string
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
	CreateAccount(CreateAccountReq) (*Account, error)
	Deposit(ChargeReq) error
	Withdraw(ChargeReq) error
	Balance(BalanceReq) (decimal.Decimal, error)
	Statement(io.Writer, StatementReq) error
}

func NewService(
	repo Repository,
	system_accts map[string]snowflake.ID,
) (*serviceImpl, error) {
	for c, id := range system_accts {
		a, err := repo.GetAcct(id)
		if err != nil {
			return nil, err
		}
		if a.currency != c {
			return nil, fmt.Errorf("provided system account %v for currency %s does not match records", id, c)
		}
	}

	// hardcoded for "simplicity", but this should be retrieved from
	// the node environment like some EC2 identifier
	node, err := snowflake.NewNode(123456789)
	if err != nil {
		return nil, err
	}
	svc := &serviceImpl{
		repo:         repo,
		system_accts: system_accts,
		node:         node,
	}

	return svc, nil
}

var (
	_ Service = (*serviceImpl)(nil)
)

type serviceImpl struct {
	repo         Repository
	system_accts map[string]snowflake.ID
	node         *snowflake.Node
}

func (s *serviceImpl) CreateAccount(req CreateAccountReq) (*Account, error) {
	req.AcctID = s.node.Generate()
	err := s.repo.CreateAccount(req)
	if err != nil {
		return nil, err
	}

	acct := &Account{
		acctID: req.AcctID,
	}
	return acct, err
}

func (s *serviceImpl) Deposit(req ChargeReq) error {
	sysAcct := s.system_accts[req.Currency]
	err := s.repo.CreditUser(req.Amount, req.AcctID, sysAcct)
	return err
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
