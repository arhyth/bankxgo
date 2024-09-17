package main

import (
	"net/http"
	"os"

	"github.com/arhyth/bankxgo"

	"github.com/rs/zerolog"
)

// edit the following main block so that connstr is derived from a viper configuration
func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()
	pgendpt, err := bankxgo.NewPostgresEndpoint(connStr)
	if err != nil {
		logger.Err(err).Msg("error starting database")
	}
	svc, err := bankxgo.NewService(pgendpt, sysAccts)
	if err != nil {
		logger.Err(err).Msg("error starting service")
	}
	hndlr := bankxgo.NewHTTPHandler(svc, &logger)

	http.ListenAndServe(":3000", hndlr)
}
