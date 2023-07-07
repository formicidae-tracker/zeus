package main

import (
	"os"

	"github.com/formicidae-tracker/olympus/pkg/tm"
	"github.com/formicidae-tracker/zeus/internal/zeus"
	"github.com/jessevdk/go-flags"
)

type Options struct {
}

var opts = &Options{}

var parser = flags.NewParser(opts, flags.Default)

func Execute() error {
	if _, err := parser.Parse(); err != nil {
		return err
	}

	return nil
}

func main() {
	setUpTelemetry()

	if err := Execute(); err != nil {
		if ferr, ok := err.(*flags.Error); ok == true && ferr.Type == flags.ErrHelp {
			return
		}
		os.Exit(1)
	}
}

func setUpTelemetry() {
	otel := os.Getenv("ZEUS_CLI_OTEL_ENDPOINT")
	if len(otel) == 0 {
		return
	}
	tm.SetUpTelemetry(tm.OtelProviderArgs{
		CollectorURL:         otel,
		ServiceName:          "zeus-cli",
		ServiceVersion:       zeus.ZEUS_VERSION,
		ForceFlushOnShutdown: true,
	})
}
