package main

import (
	"github.com/goosefans/mastodon-json-blocklist/internal/config"
	"github.com/goosefans/mastodon-json-blocklist/internal/data"
	"github.com/goosefans/mastodon-json-blocklist/internal/mastodon"
	"github.com/goosefans/mastodon-json-blocklist/internal/task"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"os/signal"
	"time"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	// Set up the logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if cfg.IsDevEnv() {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
		log.Warn().Msg("The service was started in development mode. Please change the 'ENVIRONMENT' variable to 'prod' in production!")
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		logLevel, err := zerolog.ParseLevel(cfg.LogLevel)
		if err != nil {
			log.Warn().Msg("An invalid log level was provided. Using the 'info' fallback value.")
			logLevel = zerolog.InfoLevel
		}
		zerolog.SetGlobalLevel(logLevel)
	}

	client := &mastodon.Client{
		URL:         cfg.MastodonBaseURL,
		AccessToken: cfg.MastodonAccessToken,
	}

	// Start the synchronization worker
	workerInterval, err := time.ParseDuration(cfg.TaskInterval)
	if err != nil {
		log.Fatal().Err(err).Msg("Invalid task interval.")
	}
	log.Info().Str("interval", workerInterval.String()).Msg("Starting synchronization worker...")
	worker := &task.RepeatingTask{
		Interval:   workerInterval,
		RunAtStart: true,
		Action: func() {
			log.Debug().Msg("Syncing data...")
			defer log.Debug().Msg("Finished syncing data.")
			raw, err := data.Retrieve(cfg.JSONUrl)
			if err != nil {
				log.Err(err).Msg("Could not retrieve/parse JSON data.")
				return
			}
			if err := client.SyncData(raw); err != nil {
				log.Err(err).Msg("Could not sync data.")
			}
		},
	}
	worker.Start()
	defer worker.Stop()

	// Wait for a Ctrl-C signal
	log.Info().Msg("The application has been started. To stop it press Ctrl-C.")
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt)
	<-shutdown
}
