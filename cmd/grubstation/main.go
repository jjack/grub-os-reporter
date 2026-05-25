package main

import (
	"github.com/rs/zerolog/log"
	"os"

	"github.com/jjack/grubstation/internal/cli"
)

func main() {
	app := cli.NewCLI()
	if err := app.Execute(); err != nil {
		log.Error().Err(err).Msg("Error executing command")
		os.Exit(1)
	}
}
