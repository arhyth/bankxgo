package main

import (
	"flag"
	"net/http"
	"os"

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

	pgendpt, err := bankxgo.NewPostgresEndpoint(cfg.Database.ConnectionString, &logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("error starting database")
	}

	sysAccts := make(map[string]snowflake.ID)
	for c, sa := range cfg.Database.SystemAccounts {
		id, err := snowflake.ParseString(sa)
		if err != nil {
			logger.Fatal().
				Err(err).
				Str("currency", c).
				Msg("error parsing system account ID")
		}
		sysAccts[c] = id
	}

	svc, err := bankxgo.NewService(pgendpt, sysAccts, &logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("error starting service")
	}
	mw := bankxgo.NewValidationMiddleware(pgendpt, sysAccts)
	svc = mw(svc)
	hndlr := bankxgo.NewHTTPHandler(svc, &logger)

	http.ListenAndServe(":3000", hndlr)
}
