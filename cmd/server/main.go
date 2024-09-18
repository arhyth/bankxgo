package main

import (
	"flag"
	"net/http"
	"os"
	"strings"

	"github.com/arhyth/bankxgo"
	"github.com/bwmarrin/snowflake"
	"gopkg.in/yaml.v3"

	"github.com/rs/zerolog"
)

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()

	var cfg bankxgo.Config
	cfp := flag.String("config", "config.yml", "path to configuration file")
	cfgfl, err := os.Open(*cfp)
	if err != nil {
		logger.Fatal().Err(err).Msg("error opening config file")
	}
	if err = yaml.NewDecoder(cfgfl).Decode(&cfg); err != nil {
		logger.Fatal().Err(err).Msg("error decoding config file")
	}

	pgendpt, err := bankxgo.NewPostgresEndpoint(cfg.Database.ConnStr, &logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("error starting database")
	}

	sysAccts := make(map[string]snowflake.ID)
	for c, sa := range cfg.SystemAccounts {
		id, err := snowflake.ParseString(sa)
		if err != nil {
			logger.Fatal().
				Err(err).
				Str("currency", c).
				Msg("error parsing system account ID")
		}
		sysAccts[strings.ToUpper(c)] = id
	}

	svc, err := bankxgo.NewService(pgendpt, sysAccts, &logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("error starting service")
	}

	limitmw := bankxgo.NewlimitMiddleware(&cfg.ServiceLimits)
	validmw := bankxgo.NewValidationMiddleware(pgendpt, sysAccts)
	// !!! note: the order of middlewares is inverse of the call order
	mws := []bankxgo.Middleware{
		validmw,
		limitmw,
	}
	for _, mw := range mws {
		svc = mw(svc)
	}
	hndlr := bankxgo.NewHTTPHandler(svc, &logger)

	http.ListenAndServe(":3000", hndlr)
}
