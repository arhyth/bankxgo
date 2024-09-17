package bankxgo

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/bwmarrin/snowflake"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
)

var (
	statusOK = []byte(`{"status":"OK"}`)
)

type balanceJSONResp struct {
	Balance decimal.Decimal `json:"balance"`
}

func NewHTTPHandler(svc Service, log *zerolog.Logger) http.Handler {
	hndlr := &httpHandler{
		Svc: svc,
		Log: log,
	}
	mux := chi.NewMux()
	mux.NotFound(HTTPNotFound)
	mux.Route("/accounts", func(r chi.Router) {
		r.Route("/{acctID:[0-9]+}", func(rr chi.Router) {
			rr.Post("/deposit", hndlr.Deposit)
			rr.Post("/withdraw", hndlr.Withdraw)
			rr.Get("/balance", hndlr.Balance)
			rr.Get("/statement", hndlr.Statement)
		})
	})

	return mux
}

type httpHandler struct {
	Svc Service
	Log *zerolog.Logger
}

func (h *httpHandler) Deposit(w http.ResponseWriter, r *http.Request) {
	buf, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		h.Log.Err(err).Str("method", "deposit").Msg("error reading HTTP request")
		WriteHTTPError(w, ErrInternalServer)
		return
	}
	var req ChargeReq
	if err = json.Unmarshal(buf, &req); err != nil {
		h.Log.Err(err).Str("method", "deposit").Msg("error unmarshalling JSON")
		WriteHTTPError(w, ErrBadRequest{Fields: map[string]string{"request body": "malformed JSON"}})
		return
	}
	pid := chi.URLParam(r, "acctID")
	acctID, err := snowflake.ParseString(pid)
	if err != nil {
		h.Log.Err(err).Str("method", "deposit").Msg("error parsing account ID")
		WriteHTTPError(w, ErrBadRequest{map[string]string{"acctID": "invalid format"}})
		return
	}
	req.AcctID = acctID
	bal, err := h.Svc.Deposit(req)
	if err != nil {
		WriteHTTPError(w, err)
		return
	}

	resp := balanceJSONResp{Balance: *bal}
	w.Header().Set("Content-Type", "application/json")
	if err = json.NewEncoder(w).Encode(resp); err != nil {
		WriteHTTPError(w, err)
	}
}

func (h *httpHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	buf, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		h.Log.Err(err).Str("method", "withdraw").Msg("error reading HTTP request")
		WriteHTTPError(w, ErrInternalServer)
		return
	}
	var req ChargeReq
	if err = json.Unmarshal(buf, &req); err != nil {
		h.Log.Err(err).Str("method", "withdraw").Msg("error unmarshalling JSON")
		WriteHTTPError(w, ErrBadRequest{Fields: map[string]string{"request body": "malformed JSON"}})
		return
	}
	pid := chi.URLParam(r, "acctID")
	acctID, err := snowflake.ParseString(pid)
	if err != nil {
		h.Log.Err(err).Str("method", "withdraw").Msg("error parsing account ID")
		WriteHTTPError(w, ErrBadRequest{map[string]string{"acctID": "invalid format"}})
		return
	}
	req.AcctID = acctID
	bal, err := h.Svc.Withdraw(req)
	if err != nil {
		WriteHTTPError(w, err)
		return
	}

	resp := balanceJSONResp{Balance: *bal}
	w.Header().Set("Content-Type", "application/json")
	if err = json.NewEncoder(w).Encode(resp); err != nil {
		WriteHTTPError(w, err)
	}
}

func (h *httpHandler) Balance(w http.ResponseWriter, r *http.Request) {
	email := r.Header.Get("email")
	if email == "" {
		h.Log.Error().Str("method", "balance").Msg("missing/invalid email")
		WriteHTTPError(w, ErrBadRequest{map[string]string{"email": "missing or invalid"}})
		return
	}

	pid := chi.URLParam(r, "acctID")
	acctID, err := snowflake.ParseString(pid)
	if err != nil {
		h.Log.Err(err).Str("method", "balance").Msg("error parsing account ID")
		WriteHTTPError(w, ErrBadRequest{map[string]string{"acctID": "invalid format"}})
		return
	}
	req := BalanceReq{
		AcctID: acctID,
		Email:  email,
	}
	bal, err := h.Svc.Balance(req)
	if err != nil {
		WriteHTTPError(w, err)
		return
	}

	resp := balanceJSONResp{Balance: *bal}
	w.Header().Set("Content-Type", "application/json")
	if err = json.NewEncoder(w).Encode(resp); err != nil {
		WriteHTTPError(w, err)
	}
}

func (h *httpHandler) Statement(w http.ResponseWriter, r *http.Request) {
	email := r.Header.Get("email")
	if email == "" {
		h.Log.Error().Str("method", "statement").Msg("missing/invalid email")
		WriteHTTPError(w, ErrBadRequest{map[string]string{"email": "missing or invalid"}})
		return
	}
	pid := chi.URLParam(r, "acctID")
	acctID, err := snowflake.ParseString(pid)
	if err != nil {
		h.Log.Err(err).Str("method", "statement").Msg("error parsing account ID")
		WriteHTTPError(w, ErrBadRequest{map[string]string{"acctID": "invalid format"}})
		return
	}
	req := StatementReq{
		AcctID: acctID,
		Email:  email,
	}
	if err = h.Svc.Statement(w, req); err != nil {
		WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	_, err = w.Write(statusOK)
	if err != nil {
		WriteHTTPError(w, err)
	}
}

func WriteHTTPError(w http.ResponseWriter, err error) {
	var ne error
	defer func() {
		if ne != nil {
			log.Error().
				Err(ne).
				Msg("error response encoding failed")
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	errnf := &ErrNotFound{}
	errbr := &ErrBadRequest{}
	if errors.As(err, errnf) {
		w.WriteHeader(http.StatusNotFound)
		ne = json.NewEncoder(w).Encode(errnf)
	} else if errors.As(err, errbr) {
		w.WriteHeader(http.StatusBadRequest)
		ne = json.NewEncoder(w).Encode(errbr)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		resp := map[string]string{
			"message": "server error",
		}
		ne = json.NewEncoder(w).Encode(resp)
	}
}

func HTTPNotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	w.Header().Set("Content-Type", "application/json")
	resp := map[string]string{
		"path": r.URL.Path,
	}
	json.NewEncoder(w).Encode(resp)
}
