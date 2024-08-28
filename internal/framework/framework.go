package framework

import (
	"github.com/kloudlite/kloudmeter/internal/app"
	"github.com/kloudlite/kloudmeter/internal/env"
	httpServer "github.com/kloudlite/kloudmeter/pkg/http-server"
	"github.com/kloudlite/kloudmeter/pkg/logging"
	"github.com/kloudlite/kloudmeter/pkg/nats"
	"go.uber.org/fx"
)

type fm struct {
	ev *env.Env
}

var Module = fx.Module(
	"framework",

	fx.Provide(func(ev *env.Env) *fm {
		return &fm{ev}
	}),

	fx.Provide(func(ev *env.Env, logger logging.Logger) (*nats.Client, error) {
		return nats.NewClient(ev.NatsURL, nats.ClientOpts{
			Name:   ev.MeterNatsStream,
			Logger: logger,
		})
	}),

	fx.Provide(func(c *nats.Client) (*nats.JetstreamClient, error) {
		return nats.NewJetstreamClient(c)
	}),

	fx.Provide(func(logger logging.Logger, e *env.Env) httpServer.Server {
		corsOrigins := "https://studio.apollographql.com,http://localhost:3000"
		return httpServer.NewServer(httpServer.ServerArgs{Logger: logger, CorsAllowOrigins: &corsOrigins, IsDev: e.IsDev})
	}),

	fx.Invoke(func(server httpServer.Server, envVars *env.Env) error {
		return server.Listen(":" + envVars.HttpServerPort)
	}),

	app.Module,
)
