package main

import (
	"context"
	"github.com/GSH-LAN/Unwindia_common/src/go/config"
	"github.com/GSH-LAN/Unwindia_dotlan_forum_manager/cmd/unwindia_dotlan_forum_manager/environment"
	"github.com/GSH-LAN/Unwindia_dotlan_forum_manager/cmd/unwindia_dotlan_forum_manager/server"
	"github.com/gammazero/workerpool"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func main() {
	mainContext, cancel := context.WithCancel(context.Background())
	err := godotenv.Load()
	if err != nil && !strings.Contains(err.Error(), "no such file") {
		log.Fatal().Err(err).Msg("Error loading .env file")
	}

	env := environment.Get()

	var configClient config.ConfigClient
	if env.ConfigFileName != "" {
		configClient, err = config.NewConfigFile(mainContext, env.ConfigFileName, env.ConfigTemplatesDir)
	} else {
		configClient, err = config.NewConfigClient()
	}

	if err != nil {
		cancel()
		log.Fatal().Err(err).Msg("Error initializing config")
	}

	wp := workerpool.New(env.WorkerCount)

	srv, err := server.NewServer(mainContext, env, configClient, wp)
	if err != nil {
		cancel()
		log.Fatal().Err(err).Msg("Error creating server")
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill, syscall.SIGTERM)
	go func() {
		<-c
		cancel()
		if err := srv.Stop(); err != nil {
			log.Error().Err(err).Msg("Error stopping server")
		}
	}()

	err = srv.Start()
	if err != nil {
		log.Fatal().Err(err).Msg("Error starting server")
	}
}
