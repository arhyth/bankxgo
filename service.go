package bankxgo

import (
	"io"

	"github.com/bwmarrin/snowflake"
	"github.com/shopspring/decimal"
)

type Account struct {
	// AcctID and other instances of it in struct fields in this package
	// refers to the public id of the account (as opposed to its BIGINT id)
	// hence named `pub_id` column in the database
	AcctID   snowflake.ID
	Email    string
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
	Deposit(ChargeReq) (*decimal.Decimal, error)
	Withdraw(ChargeReq) (*decimal.Decimal, error)
	Balance(BalanceReq) (*decimal.Decimal, error)
	Statement(io.Writer, StatementReq) error
}

func NewService(
	repo Repository,
	sysAccts map[string]snowflake.ID,
) (*serviceImpl, error) {
	for c, id := range sysAccts {
		a, err := repo.GetAccount(id)
		if err != nil {
			return nil, err
		}
		if a.Currency != c {
			return nil, ErrNotFound{ID: id.Int64()}
		}
	}

	// hardcoded for "simplicity", but in a real world service this should be
	// seeded with data from the node environment, ie., EC2 identifier
	node, err := snowflake.NewNode(888)
	if err != nil {
		return nil, err
	}
	svc := &serviceImpl{
		repo:     repo,
		sysAccts: sysAccts,
		node:     node,
	}
	return svc, nil
}

var (
	_ Service = (*serviceImpl)(nil)
)

type serviceImpl struct {
	repo     Repository
	sysAccts map[string]snowflake.ID
	node     *snowflake.Node
}

func (s *serviceImpl) CreateAccount(req CreateAccountReq) (*Account, error) {
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

func (s *serviceImpl) Deposit(req ChargeReq) (*decimal.Decimal, error) {
	bal, err := s.repo.CreditUser(req.Amount, req.AcctID, s.sysAccts[req.Currency])
	return bal, err
}

func (s *serviceImpl) Withdraw(req ChargeReq) (*decimal.Decimal, error) {
	bal, err := s.repo.DebitUser(req.Amount, req.AcctID, s.sysAccts[req.Currency])
	return bal, err
}

func (s *serviceImpl) Balance(req BalanceReq) (*decimal.Decimal, error) {
	acct, err := s.repo.GetAccount(req.AcctID)
	if err != nil {
		return nil, err
	}
	bal := acct.Balance
	return &bal, err
}

func (s *serviceImpl) Statement(w io.Writer, req StatementReq) error {
	return nil
}
