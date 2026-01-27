// Package main - Scala Application.main + ImporterSource equivalent.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"sbx-erp-whmovement-go/internal/config"
	"sbx-erp-whmovement-go/internal/streams"
)

func main() {
	// JSON logger (Scala log4j2.xml)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})

	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Config failed")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	processor := streams.NewProcessor(cfg)

	// Graceful shutdown
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sig
		log.Info().Msg("Shutdown signal")
		cancel()
	}()

	log.Info().
		Str("bucket", cfg.S3Bucket).
		Str("soccod", cfg.SocCod).
		Msg("Starting importer")

	if err := processor.Run(ctx); err != nil {
		log.Fatal().Err(err).Msg("Processor failed")
	}
	log.Info().Msg("Completed")
}
