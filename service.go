package bankxgo

import (
	"fmt"
	"io"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/go-pdf/fpdf"
	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"
)

type Account struct {
	// AcctID and other instances of it in struct fields in this package
	// refers to the public id of the account (as opposed to its BIGINT id)
	// hence named `pub_id` column in the database
	AcctID   snowflake.ID    `json:"acctID"`
	Email    string          `json:"-"`
	Currency string          `json:"-"`
	Balance  decimal.Decimal `json:"-"`
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
	log *zerolog.Logger,
) (Service, error) {
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
		log:      log,
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
	log      *zerolog.Logger
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
	if err != nil {
		s.log.Error().Err(err).Msg("Deposit failed")
		return nil, err
	}
	return bal, err
}

func (s *serviceImpl) Withdraw(req ChargeReq) (*decimal.Decimal, error) {
	bal, err := s.repo.DebitUser(req.Amount, req.AcctID, s.sysAccts[req.Currency])
	if err != nil {
		s.log.Error().Err(err).Msg("Withdraw failed")
		return nil, err
	}
	return bal, err
}

func (s *serviceImpl) Balance(req BalanceReq) (*decimal.Decimal, error) {
	acct, err := s.repo.GetAccount(req.AcctID)
	if err != nil {
		s.log.Error().Err(err).Msg("Balance failed")
		return nil, err
	}
	bal := acct.Balance
	return &bal, err
}

type Charge struct {
	Amount    decimal.Decimal
	Typ       string
	CreatedAt time.Time
}

func (s *serviceImpl) Statement(w io.Writer, req StatementReq) error {
	charges, err := s.repo.GetAccountCharges(req.AcctID)
	if err != nil {
		s.log.Error().Err(err).Msg("Statement failed")
		return err
	}

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(190, 10, "Statement of Account", "", 1, "C", false, 0, "")

	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(25, 8, "Account ID:", "", 0, "L", false, 0, "")
	pdf.Cell(2, 8, "")
	pdf.SetFillColor(211, 212, 208)
	pdf.SetFont("Arial", "", 12)
	pdf.CellFormat(50, 8, req.AcctID.String(), "", 1, "R", true, 0, "")
	pdf.Ln(6)

	tableHeader(pdf)

	balance := decimal.New(0, 0)
	creditStr, debitStr, dateStr := "", "", ""
	var (
		isFirstPage bool
		lineCount   int
	)
	isFirstPage = true
	cl := len(charges)
	for ci, charge := range charges {
		// skip last line and process separately for estetik ;)
		if ci == cl-1 {
			break
		}
		if lineCount == 30 && isFirstPage {
			pdf.AddPage()
			pdf.Ln(5)
			tableHeader(pdf)
			lineCount = 0
			isFirstPage = false
		}
		if lineCount == 35 {
			pdf.AddPage()
			pdf.Ln(5)
			tableHeader(pdf)
			lineCount = 0
		}
		dateStr = charge.CreatedAt.Format("2006-01-02")
		if charge.Typ == "credit" {
			debitStr = ""
			creditStr = charge.Amount.StringFixed(2)
			balance = balance.Add(charge.Amount)
		} else {
			creditStr = ""
			debitStr = charge.Amount.StringFixed(2)
			balance = balance.Sub(charge.Amount)
		}
		pdf.Cell(20, 6, "")
		pdf.CellFormat(30, 6, dateStr, "", 0, "C", false, 0, "")
		pdf.CellFormat(40, 6, debitStr, "", 0, "C", false, 0, "")
		pdf.CellFormat(40, 6, creditStr, "", 0, "C", false, 0, "")
		pdf.CellFormat(40, 6, balance.StringFixed(2), "", 1, "C", false, 0, "")
		pdf.Ln(1)

		lineCount += 1
	}

	// process last line
	charge := charges[cl-1]
	dateStr = charge.CreatedAt.Format("2006-01-02")
	if charge.Typ == "credit" {
		debitStr = ""
		creditStr = charge.Amount.StringFixed(2)
		balance = balance.Add(charge.Amount)
	} else {
		creditStr = ""
		debitStr = charge.Amount.StringFixed(2)
		balance = balance.Sub(charge.Amount)
	}
	pdf.Cell(20, 6, "")
	pdf.CellFormat(30, 6, dateStr, "", 0, "C", false, 0, "")
	pdf.CellFormat(40, 6, debitStr, "", 0, "C", false, 0, "")
	pdf.CellFormat(40, 6, creditStr, "", 0, "C", false, 0, "")
	pdf.SetFillColor(140, 212, 130)
	pdf.CellFormat(40, 6, balance.StringFixed(2), "", 1, "C", true, 0, "")

	err = pdf.Output(w)
	if err != nil {
		s.log.
			Error().
			Err(fmt.Errorf("pdf.Output: %w", err)).
			Msg("Statement failed")
	}

	return err
}

func tableHeader(pdf *fpdf.Fpdf) {
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(20, 10, "")
	pdf.CellFormat(30, 10, "Date", "", 0, "C", false, 0, "")
	pdf.CellFormat(40, 10, "Debit", "", 0, "C", false, 0, "")
	pdf.CellFormat(40, 10, "Credit", "", 0, "C", false, 0, "")
	pdf.CellFormat(40, 10, "Balance", "", 1, "C", false, 0, "")
	pdf.Ln(1)
	pdf.SetFont("Arial", "", 10)
}
