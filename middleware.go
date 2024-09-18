package bankxgo

import (
	"errors"
	"io"
	"regexp"

	"github.com/bwmarrin/snowflake"
	"github.com/shopspring/decimal"
	"github.com/sony/gobreaker/v2"
	"golang.org/x/sync/semaphore"
)

var (
	emailRegex = regexp.MustCompile(`^[\w\.-]+@[a-zA-Z\d\.-]+\.[a-zA-Z]{2,}$`)
)

var _ Service = (*validationMiddleware)(nil)

type Middleware func(Service) Service

// validationMiddleware validates the following invariants:
// 1. The account exists in the repository [Withdraw, Deposit, Balance, Statement]
// 2. The account is not a system acount [Withdraw, Deposit]
// 3. The account ID and email belong to the same account [Withdraw, Deposit, Balance, Statement]
// 4. The currency is supported, ie. there exist a system account for it [CreateAccount]
// 5. The email is of valid format [CreateAccount]
// 6. The amount is not negative [Deposit, Withdraw]
// 7. The account has sufficient balance for withdrawal [Withdraw]
type validationMiddleware struct {
	next     Service
	repo     Repository
	sysAccts map[string]snowflake.ID
}

func (v *validationMiddleware) CreateAccount(req CreateAccountReq) (*Account, error) {
	if !emailRegex.MatchString(req.Email) {
		return nil, ErrBadRequest{Fields: map[string]string{"email": "invalid"}}
	}
	if _, exists := v.sysAccts[req.Currency]; !exists {
		return nil, ErrBadRequest{Fields: map[string]string{"currency": "unsupported"}}
	}
	return v.next.CreateAccount(req)
}

func (v *validationMiddleware) Deposit(req ChargeReq) (*decimal.Decimal, error) {
	if req.Amount.IsNegative() {
		return nil, ErrBadRequest{Fields: map[string]string{"amount": "negative"}}
	}

	for _, id := range v.sysAccts {
		if id == req.AcctID {
			return nil, ErrBadRequest{Fields: map[string]string{"acctID": "system account not allowed"}}
		}
	}

	acct, err := v.repo.GetAccount(req.AcctID)
	errnf := ErrNotFound{}
	if err != nil && errors.As(err, &errnf) {
		return nil, errnf
	}
	if acct.Email != req.Email {
		return nil, ErrBadRequest{Fields: map[string]string{"email": "mismatch"}}
	}
	// this should not happen unless a system account for the currency is removed
	if _, exists := v.sysAccts[acct.Currency]; !exists {
		return nil, ErrInternalServer
	}
	req.Currency = acct.Currency

	return v.next.Deposit(req)
}

func (v *validationMiddleware) Withdraw(req ChargeReq) (*decimal.Decimal, error) {
	if req.Amount.IsNegative() {
		return nil, ErrBadRequest{Fields: map[string]string{"amount": "negative"}}
	}

	for _, id := range v.sysAccts {
		if id == req.AcctID {
			return nil, ErrBadRequest{Fields: map[string]string{"acctID": "system account not allowed"}}
		}
	}

	acct, err := v.repo.GetAccount(req.AcctID)
	errnf := ErrNotFound{}
	if err != nil && errors.As(err, &errnf) {
		return nil, errnf
	}
	if acct.Email != req.Email {
		return nil, ErrBadRequest{Fields: map[string]string{"email": "mismatch"}}
	}
	if acct.Balance.LessThan(req.Amount) {
		return nil, ErrBadRequest{Fields: map[string]string{"amount": "insufficient balance"}}
	}
	// this should not happen unless a system account for the currency is removed
	if _, exists := v.sysAccts[acct.Currency]; !exists {
		return nil, ErrInternalServer
	}
	req.Currency = acct.Currency

	return v.next.Withdraw(req)
}

func (v *validationMiddleware) Balance(req BalanceReq) (*decimal.Decimal, error) {
	acct, err := v.repo.GetAccount(req.AcctID)
	errnf := ErrNotFound{}
	if err != nil && errors.As(err, &errnf) {
		return nil, errnf
	}
	if acct.Email != req.Email {
		return nil, ErrBadRequest{Fields: map[string]string{"email": "mismatch"}}
	}

	return v.next.Balance(req)
}

func (v *validationMiddleware) Statement(w io.Writer, req StatementReq) error {
	acct, err := v.repo.GetAccount(req.AcctID)
	errnf := ErrNotFound{}
	if err != nil && errors.As(err, &errnf) {
		return errnf
	}
	if acct.Email != req.Email {
		return ErrBadRequest{Fields: map[string]string{"email": "mismatch"}}
	}

	return v.next.Statement(w, req)
}

func NewValidationMiddleware(repo Repository, sysAccts map[string]snowflake.ID) Middleware {
	return func(svc Service) Service {
		return &validationMiddleware{
			next:     svc,
			repo:     repo,
			sysAccts: sysAccts,
		}
	}
}

//
// Rate limiting middlewares
//

// limitMiddleware limits the number of in-flight requests to the service by using
// a weighted semaphore, i.e., x/sync/semaphore.Semaphore with an acquisition timeout.
// As limits are static and servers may be deployed to a heterogeneous set of machines,
// hence, having to manually tune limits for each server, this solution is something
// likely implemented very differently in a real-world application, but it is a good
// example of load shedding.
type limitMiddleware struct {
	next   Service
	limits *ServiceLimits
}

var _ Service = (*limitMiddleware)(nil)

type ServiceLimits struct {
	CreateAccount *semaphore.Weighted
	Deposit       *semaphore.Weighted
	Withdraw      *semaphore.Weighted
	Balance       *semaphore.Weighted
	Statement     *semaphore.Weighted
}

func NewlimitMiddleware(limits *ServiceLimits) Middleware {
	return func(next Service) Service {
		return &limitMiddleware{
			next:   next,
			limits: limits,
		}
	}
}

func (l *limitMiddleware) CreateAccount(req CreateAccountReq) (*Account, error) {
	return l.next.CreateAccount(req)
}

func (l *limitMiddleware) Deposit(req ChargeReq) (*decimal.Decimal, error) {
	return l.next.Deposit(req)
}

func (l *limitMiddleware) Withdraw(req ChargeReq) (*decimal.Decimal, error) {
	return l.next.Withdraw(req)
}

func (l *limitMiddleware) Balance(req BalanceReq) (*decimal.Decimal, error) {
	return l.next.Balance(req)
}

func (l *limitMiddleware) Statement(w io.Writer, req StatementReq) error {
	return l.next.Statement(w, req)
}

type ServiceBreaker struct {
	CreateAccount *gobreaker.TwoStepCircuitBreaker[*Account]
	Deposit       *gobreaker.TwoStepCircuitBreaker[*decimal.Decimal]
	Withdraw      *gobreaker.TwoStepCircuitBreaker[*decimal.Decimal]
	Balance       *gobreaker.TwoStepCircuitBreaker[*decimal.Decimal]
	Statement     *gobreaker.TwoStepCircuitBreaker[interface{}]
}

// circuitBreakMiddleware is a middleware that implements the circuit breaker pattern.
// It works in conjunction with limitMiddleware to limit the number of in-flight
// requests to the service when the circuit is not in `closed` state, i.e., the service
// is experiencing heavy load and is struggling to release tokens from the limit
// semaphores within request deadline
type circuitBreakMiddleware struct {
	next  Service
	brkrs *ServiceBreaker
}

var _ Service = (*circuitBreakMiddleware)(nil)

func NewCircuitBreakMiddleware(brkrs *ServiceBreaker) Middleware {
	return func(next Service) Service {
		return &circuitBreakMiddleware{
			next:  next,
			brkrs: brkrs,
		}
	}
}

func (c *circuitBreakMiddleware) CreateAccount(req CreateAccountReq) (*Account, error) {
	return c.next.CreateAccount(req)
}

func (c *circuitBreakMiddleware) Deposit(req ChargeReq) (*decimal.Decimal, error) {
	return c.next.Deposit(req)
}

func (c *circuitBreakMiddleware) Withdraw(req ChargeReq) (*decimal.Decimal, error) {
	return c.next.Withdraw(req)
}

func (c *circuitBreakMiddleware) Balance(req BalanceReq) (*decimal.Decimal, error) {
	return c.next.Balance(req)
}

func (c *circuitBreakMiddleware) Statement(w io.Writer, req StatementReq) error {
	return c.next.Statement(w, req)
}
