package bankxgo

import (
	"github.com/bwmarrin/snowflake"
	"github.com/shopspring/decimal"
)

type Repository interface {
	CreateAccount(req CreateAccountReq) error
	CreditUser(amount decimal.Decimal, userAcct, systemAcct snowflake.ID) error
	DebitUser(amount decimal.Decimal, userAcct, systemAcct snowflake.ID) error
	GetAcct(id snowflake.ID) (*Account, error)
}
