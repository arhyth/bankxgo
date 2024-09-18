package bankxgo

import (
	"context"
	"io"
	"regexp"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/shopspring/decimal"
	"golang.org/x/time/rate"
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
	if req.Email == "" {
		return nil, ErrBadRequest{Fields: map[string]string{"email": "missing/invalid"}}
	}

	for _, id := range v.sysAccts {
		if id == req.AcctID {
			return nil, ErrBadRequest{Fields: map[string]string{"acctID": "system account not allowed"}}
		}
	}

	acct, err := v.repo.GetAccount(req.AcctID)
	if err != nil {
		return nil, err
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
	if req.Email == "" {
		return nil, ErrBadRequest{Fields: map[string]string{"email": "missing/invalid"}}
	}

	for _, id := range v.sysAccts {
		if id == req.AcctID {
			return nil, ErrBadRequest{Fields: map[string]string{"acctID": "system account not allowed"}}
		}
	}

	acct, err := v.repo.GetAccount(req.AcctID)
	if err != nil {
		return nil, err
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
	if req.Email == "" {
		return nil, ErrBadRequest{Fields: map[string]string{"email": "missing/invalid"}}
	}
	acct, err := v.repo.GetAccount(req.AcctID)
	if err != nil {
		return nil, err
	}
	if acct.Email != req.Email {
		return nil, ErrBadRequest{Fields: map[string]string{"email": "mismatch"}}
	}

	return v.next.Balance(req)
}

func (v *validationMiddleware) Statement(w io.Writer, req StatementReq) error {
	if req.Email == "" {
		return ErrBadRequest{Fields: map[string]string{"email": "missing/invalid"}}
	}
	acct, err := v.repo.GetAccount(req.AcctID)
	if err != nil {
		return err
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
	limits *serviceLimits
}

var _ Service = (*limitMiddleware)(nil)

// endpointLimit defines the deadline/SLO and a token bucket rate limiter
// for a service endpoint
type endpointLimit struct {
	Slo time.Duration
	Lmt *rate.Limiter
}

type serviceLimits struct {
	CreateAccount *endpointLimit
	Deposit       *endpointLimit
	Withdraw      *endpointLimit
	Balance       *endpointLimit
	Statement     *endpointLimit
}

func NewlimitMiddleware(cfg *ServiceLimitsCfg) Middleware {
	limits := &serviceLimits{
		CreateAccount: &endpointLimit{
			Slo: time.Duration(cfg.CreateAccount.SloMs) * time.Millisecond,
			Lmt: rate.NewLimiter(rate.Limit(cfg.CreateAccount.Rate), cfg.CreateAccount.Burst),
		},
		Deposit: &endpointLimit{
			Slo: time.Duration(cfg.Deposit.SloMs) * time.Millisecond,
			Lmt: rate.NewLimiter(rate.Limit(cfg.Deposit.Rate), cfg.Deposit.Burst),
		},
		Withdraw: &endpointLimit{
			Slo: time.Duration(cfg.Withdraw.SloMs) * time.Millisecond,
			Lmt: rate.NewLimiter(rate.Limit(cfg.Withdraw.Rate), cfg.Withdraw.Burst),
		},
		Balance: &endpointLimit{
			Slo: time.Duration(cfg.Balance.SloMs) * time.Millisecond,
			Lmt: rate.NewLimiter(rate.Limit(cfg.Balance.Rate), cfg.Balance.Burst),
		},
		Statement: &endpointLimit{
			Slo: time.Duration(cfg.Statement.SloMs) * time.Millisecond,
			Lmt: rate.NewLimiter(rate.Limit(cfg.Statement.Rate), cfg.Statement.Burst),
		},
	}
	return func(next Service) Service {
		return &limitMiddleware{
			next:   next,
			limits: limits,
		}
	}
}

func (l *limitMiddleware) CreateAccount(req CreateAccountReq) (*Account, error) {
	ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(l.limits.CreateAccount.Slo))
	if err := l.limits.CreateAccount.Lmt.Wait(ctx); err != nil {
		return nil, ErrServiceUnavailable
	}
	return l.next.CreateAccount(req)
}

func (l *limitMiddleware) Deposit(req ChargeReq) (*decimal.Decimal, error) {
	ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(l.limits.Deposit.Slo))
	if err := l.limits.Deposit.Lmt.Wait(ctx); err != nil {
		return nil, ErrServiceUnavailable
	}
	return l.next.Deposit(req)
}

func (l *limitMiddleware) Withdraw(req ChargeReq) (*decimal.Decimal, error) {
	ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(l.limits.Withdraw.Slo))
	if err := l.limits.Withdraw.Lmt.Wait(ctx); err != nil {
		return nil, ErrServiceUnavailable
	}
	return l.next.Withdraw(req)
}

func (l *limitMiddleware) Balance(req BalanceReq) (*decimal.Decimal, error) {
	ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(l.limits.Balance.Slo))
	if err := l.limits.Balance.Lmt.Wait(ctx); err != nil {
		return nil, ErrServiceUnavailable
	}
	return l.next.Balance(req)
}

func (l *limitMiddleware) Statement(w io.Writer, req StatementReq) error {
	ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(l.limits.Statement.Slo))
	if err := l.limits.Statement.Lmt.Wait(ctx); err != nil {
		return ErrServiceUnavailable
	}
	return l.next.Statement(w, req)
}
