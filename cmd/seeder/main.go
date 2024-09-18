package main

import (
	"flag"
	"os"

	"github.com/arhyth/bankxgo"
	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"
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

	lh, err := bankxgo.NewLocalHelper(&cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("error starting local helper")
	}
	if _, err = lh.InitDB(); err != nil {
		logger.Fatal().Err(err).Msg("error initializing database")
	}
	if err = lh.PrepareSystemAccounts(); err != nil {
		logger.Fatal().Err(err).Msg("error preparing system accounts")
	}
}
