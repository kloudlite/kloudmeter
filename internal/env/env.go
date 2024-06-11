package env

import (
	"github.com/codingconcepts/env"
	"github.com/kloudlite/kloudmeter/pkg/errors"
)

type Env struct {
	NatsURL         string `env:"NATS_URL" required:"true" default:"nats://localhost:4222"`
	MeterNatsStream string `env:"METER_NATS_STREAM" required:"true" default:"meters"`
	HttpServerPort  string `env:"HTTP_SERVER_PORT" required:"true" default:"8080"`
	IsDev           bool
}

func LoadEnv() (*Env, error) {
	var ev Env
	if err := env.Set(&ev); err != nil {
		return nil, errors.NewE(err)
	}
	return &ev, nil
}
