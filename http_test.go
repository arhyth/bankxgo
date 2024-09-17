package bankxgo_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/arhyth/bankxgo"
	"github.com/arhyth/bankxgo/mocks"
)

func TestHTTPDeposit(t *testing.T) {
	nooplog := zerolog.Nop()

	t.Run("Deposit returns OK on success", func(tt *testing.T) {
		as := assert.New(tt)
		ctrl := gomock.NewController(tt)
		svc := mocks.NewMockService(ctrl)
		bal := decimal.NewFromInt(1234)
		svc.EXPECT().
			Deposit(gomock.AssignableToTypeOf(bankxgo.ChargeReq{})).
			DoAndReturn(func(r bankxgo.ChargeReq) (*decimal.Decimal, error) {
				return &bal, nil
			}).
			Times(1)

		hndlr := bankxgo.NewHTTPHandler(svc, &nooplog)
		body := bytes.NewBufferString(`{"amount":1234.00}`)
		req := httptest.NewRequest(http.MethodPost, "/accounts/1834563581361305763/deposit", body)
		req.Header.Set("email", "arhyth@gmail.com")
		w := httptest.NewRecorder()
		hndlr.ServeHTTP(w, req)

		as.Equal(http.StatusOK, w.Code)
		resp := map[string]string{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		as.Nil(err)
		as.Contains(resp, "balance")
		as.Equal(resp["balance"], "1234")
	})

	t.Run("/accounts/{acctID}/deposit returns error on invalid account ID", func(tt *testing.T) {
		as := assert.New(tt)
		reqrd := require.New(tt)
		ctrl := gomock.NewController(t)
		svc := mocks.NewMockService(ctrl)
		hndlr := bankxgo.NewHTTPHandler(svc, &nooplog)

		body := bytes.NewBufferString(`{"amount":1234.00}`)
		req := httptest.NewRequest(http.MethodPost, "/accounts/24j24g*()/deposit", body)
		req.Header.Set("email", "rogue@one.com")
		w := httptest.NewRecorder()
		hndlr.ServeHTTP(w, req)

		as.Equal(http.StatusNotFound, w.Code)
		resp := map[string]string{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		reqrd.Nil(err)
		as.Contains(resp, "path")
	})

	t.Run("/accounts/{acctID}/deposit returns error on malformed request body", func(tt *testing.T) {
		as := assert.New(tt)
		reqrd := require.New(tt)
		ctrl := gomock.NewController(t)
		svc := mocks.NewMockService(ctrl)
		hndlr := bankxgo.NewHTTPHandler(svc, &nooplog)

		body := bytes.NewBufferString(`{"amount":1234.00`)
		req := httptest.NewRequest(http.MethodPost, "/accounts/123456789/deposit", body)
		req.Header.Set("email", "rogue@one.com")
		w := httptest.NewRecorder()
		hndlr.ServeHTTP(w, req)

		as.Equal(http.StatusBadRequest, w.Code)
		resp := map[string]map[string]string{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		reqrd.Nil(err)
		as.Contains(resp, "fields")
		as.Contains(resp["fields"], "request body")
	})
}
func TestHTTPWithdraw(t *testing.T) {
	nooplog := zerolog.Nop()
	t.Run("Withdraw returns OK on success", func(tt *testing.T) {
		as := assert.New(tt)
		ctrl := gomock.NewController(tt)
		svc := mocks.NewMockService(ctrl)
		balance := decimal.NewFromUint64(1234)
		svc.EXPECT().
			Withdraw(gomock.AssignableToTypeOf(bankxgo.ChargeReq{})).
			DoAndReturn(func(r bankxgo.ChargeReq) (*decimal.Decimal, error) {
				return &balance, nil
			}).
			Times(1)

		hndlr := bankxgo.NewHTTPHandler(svc, &nooplog)
		body := bytes.NewBufferString(`{"amount":1234.00}`)
		req := httptest.NewRequest(http.MethodPost, "/accounts/1834563581361305763/withdraw", body)
		req.Header.Set("email", "arhyth@gmail.com")
		w := httptest.NewRecorder()
		hndlr.ServeHTTP(w, req)

		as.Equal(http.StatusOK, w.Code)
		resp := map[string]string{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		as.Nil(err)
		as.Contains(resp, "balance")
		as.Equal(resp["balance"], "1234")
	})

	t.Run("/accounts/{acctID}/withdraw returns error on invalid account ID", func(tt *testing.T) {
		as := assert.New(tt)
		reqrd := require.New(tt)
		ctrl := gomock.NewController(tt)
		svc := mocks.NewMockService(ctrl)
		hndlr := bankxgo.NewHTTPHandler(svc, &nooplog)

		body := bytes.NewBufferString(`{"amount":1234.00}`)
		req := httptest.NewRequest(http.MethodPost, "/accounts/24j24g*()/withdraw", body)
		req.Header.Set("email", "rogue@one.com")
		w := httptest.NewRecorder()
		hndlr.ServeHTTP(w, req)

		as.Equal(http.StatusNotFound, w.Code)
		resp := map[string]string{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		reqrd.Nil(err)
		as.Contains(resp, "path")
	})

	t.Run("/accounts/{acctID}/withdraw returns error on malformed request body", func(tt *testing.T) {
		as := assert.New(tt)
		reqrd := require.New(tt)
		ctrl := gomock.NewController(t)
		svc := mocks.NewMockService(ctrl)
		hndlr := bankxgo.NewHTTPHandler(svc, &nooplog)

		body := bytes.NewBufferString(`{"amount":1234.00`)
		req := httptest.NewRequest(http.MethodPost, "/accounts/123456789/withdraw", body)
		req.Header.Set("email", "rogue@one.com")
		w := httptest.NewRecorder()
		hndlr.ServeHTTP(w, req)

		as.Equal(http.StatusBadRequest, w.Code)
		resp := map[string]map[string]string{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		reqrd.Nil(err)
		as.Contains(resp, "fields")
		as.Contains(resp["fields"], "request body")
	})
}

func TestHTTPBalance(t *testing.T) {
	nooplog := zerolog.Nop()
	t.Run("Balance returns balance amount", func(tt *testing.T) {
		as := assert.New(tt)
		ctrl := gomock.NewController(tt)
		svc := mocks.NewMockService(ctrl)
		balance := decimal.NewFromFloat(123.45)
		svc.EXPECT().
			Balance(gomock.AssignableToTypeOf(bankxgo.BalanceReq{})).
			DoAndReturn(func(r bankxgo.BalanceReq) (*decimal.Decimal, error) {
				return &balance, nil
			}).
			Times(1)

		hndlr := bankxgo.NewHTTPHandler(svc, &nooplog)
		req := httptest.NewRequest(http.MethodGet, "/accounts/1834563581361305763/balance", nil)
		req.Header.Set("email", "arhyth@gmail.com")
		w := httptest.NewRecorder()
		hndlr.ServeHTTP(w, req)

		as.Equal(http.StatusOK, w.Code)
		resp := map[string]string{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		as.Nil(err)
		as.Contains(resp, "balance")
		as.Equal(resp["balance"], balance.String())
	})
}
