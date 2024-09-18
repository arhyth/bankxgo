package bankxgo

import (
	"github.com/bwmarrin/snowflake"
	"github.com/shopspring/decimal"
)

type Repository interface {
	CreateAccount(req CreateAccountReq) error
	CreditUser(amount decimal.Decimal, userAcct, systemAcct snowflake.ID) (*decimal.Decimal, error)
	DebitUser(amount decimal.Decimal, userAcct, systemAcct snowflake.ID) (*decimal.Decimal, error)
	GetAccount(id snowflake.ID) (*Account, error)
	GetAccountCharges(id snowflake.ID) ([]Charge, error)
}
