package bankxgo

import (
	"fmt"
	"io"

	"github.com/bwmarrin/snowflake"
	"github.com/shopspring/decimal"
)

type Account struct {
	// AcctID and other instances of it in struct fields in this package
	// refers to the public id of the account (as opposed to its BIGINT id)
	// hence named `pub_id` column in the database
	AcctID   snowflake.ID
	Currency string
	Balance  decimal.Decimal
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
		a, err := repo.GetAccount(id)
		if err != nil {
			return nil, err
		}
		if a.Currency != c {
			return nil, fmt.Errorf("provided system account %v for currency %s does not match records", id, c)
		}
	}

	// hardcoded for "simplicity", but this should be seeded by data from
	// the node environment like an EC2 machine identifier or something
	node, err := snowflake.NewNode(888)
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
	// TODO: add middleware to check if currency is supported
	req.AcctID = s.node.Generate()
	err := s.repo.CreateAccount(req)
	if err != nil {
		return nil, err
	}

	acct := &Account{
		AcctID: req.AcctID,
	}
	return acct, err
}

func (s *serviceImpl) Deposit(req ChargeReq) error {
	// TODO: implement this check in middleware
	sysAcct, exists := s.system_accts[req.Currency]
	if !exists {
		return ErrBadRequest{Fields: map[string]string{"currency": "unsupported"}}
	}
	err := s.repo.CreditUser(req.Amount, req.AcctID, sysAcct)
	return err
}

func (s *serviceImpl) Withdraw(req ChargeReq) error {
	sysAcct := s.system_accts[req.Currency]
	err := s.repo.DebitUser(req.Amount, req.AcctID, sysAcct)
	return err
}

func (s *serviceImpl) Balance(req BalanceReq) (decimal.Decimal, error) {
	acct, err := s.repo.GetAccount(req.AcctID)
	if err != nil {
		return decimal.NewFromInt(0), err
	}
	return acct.Balance, err
}

func (s *serviceImpl) Statement(w io.Writer, req StatementReq) error {
	return nil
}
