package main

import (
	"log"
	"time"

	"github.com/deta/pc-cli/cmd"
	"github.com/getsentry/sentry-go"
)

func main() {
	err := sentry.Init(sentry.ClientOptions{
		Dsn:              "https://1f9559e40e1f45ce9304f5decf5345e8@o371570.ingest.sentry.io/4504717382844416",
		TracesSampleRate: 0.2,
	})
	if err != nil {
		log.Fatalf("sentry.Init: %s", err)
	}

	defer sentry.Flush(time.Second)

	cmd.Execute()
}
