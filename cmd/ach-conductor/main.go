// generated-from:9d2e1a7aff438bb75e877b034d21b525c8c10efee44288edf6ce935500a9fe76 DO NOT REMOVE, DO UPDATE

package main

import (
	"os"

	"github.com/moov-io/base/log"

	"github.com/moov-io/ach-conductor"
	"github.com/moov-io/ach-conductor/pkg/service"
)

func main() {
	env := &service.Environment{
		Logger: log.NewDefaultLogger().Set("app", log.String("ach-conductor")).Set("version", log.String(achconductor.Version)),
	}

	env, err := service.NewEnvironment(env)
	if err != nil {
		env.Logger.Fatal().LogErrorf("Error loading up environment: %v", err)
		os.Exit(1)
	}
	defer env.Shutdown()

	termListener := service.NewTerminationListener()

	stopServers := env.RunServers(termListener)
	defer stopServers()

	service.AwaitTermination(env.Logger, termListener)
}
