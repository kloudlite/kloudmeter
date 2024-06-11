package main

import (
	"context"
	"flag"
	"github.com/kloudlite/api/common"
	"github.com/kloudlite/kloudmeter/internal/env"
	"github.com/kloudlite/kloudmeter/internal/framework"
	"github.com/kloudlite/kloudmeter/pkg/logging"
	"go.uber.org/fx"
	"os"
	"time"
)

func main() {
	var isDev bool
	flag.BoolVar(&isDev, "dev", false, "--dev")
	flag.Parse()

	logger, err := logging.New(&logging.Options{Name: "kloud-meter", Dev: isDev})
	if err != nil {
		panic(err)
	}

	webApp := fx.New(
		fx.NopLogger,
		fx.Provide(
			func() logging.Logger {
				return logger
			},
		),
		fx.Provide(func() (*env.Env, error) {
			return env.LoadEnv()
		}),
		framework.Module,
	)

	ctx, cf := context.WithTimeout(context.Background(), 5*time.Second)
	defer cf()

	if err := webApp.Start(ctx); err != nil {
		logger.Errorf(err, "kloudmeter startup errors")
		logger.Infof("EXITING as errors encountered during startup")
		os.Exit(1)
	}

	common.PrintReadyBanner()
	<-webApp.Done()
}
