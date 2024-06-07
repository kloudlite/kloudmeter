package app

import (
	"context"
	"errors"
	"github.com/gofiber/fiber/v2"
	"github.com/kloudlite/api/pkg/grpc"
	httpServer "github.com/kloudlite/api/pkg/http-server"
	"github.com/kloudlite/api/pkg/kv"
	"github.com/kloudlite/api/pkg/logging"
	"github.com/kloudlite/api/pkg/messaging"
	msg_nats "github.com/kloudlite/api/pkg/messaging/nats"
	"github.com/kloudlite/api/pkg/nats"
	"github.com/kloudlite/kloudmeter/internal/domain"
	"github.com/kloudlite/kloudmeter/internal/domain/entities"
	"github.com/kloudlite/kloudmeter/internal/env"
	"net/http"

	"go.uber.org/fx"
)

type (
	IAMGrpcClient grpc.Client
)

var TOPIC_NAME_PREFIX = "ntfy:message.*"

type MeterConsumer messaging.Consumer

type CommsGrpcServer grpc.Server

var Module = fx.Module("app",

	fx.Provide(func(jc *nats.JetstreamClient) (meter kv.Repo[*entities.Meter], err error) {
		return kv.NewNatsKVRepo[*entities.Meter](context.TODO(), "meters", jc)
	}),
	domain.Module,

	fx.Provide(func(jc *nats.JetstreamClient, ev *env.Env, logger logging.Logger) (MeterConsumer, error) {
		topic := string(TOPIC_NAME_PREFIX)
		consumerName := "ntfy:message"
		return msg_nats.NewJetstreamConsumer(context.TODO(), jc, msg_nats.JetstreamConsumerArgs{
			Stream: ev.MeterNatsStream,
			ConsumerConfig: msg_nats.ConsumerConfig{
				Name:        consumerName,
				Durable:     consumerName,
				Description: "this consumer reads message from a subject dedicated to errors, that occurred when the resource was applied at the agent",
				FilterSubjects: []string{
					topic,
				},
			},
		})
	}),

	fx.Invoke(
		func(server httpServer.Server, envVars *env.Env, logr logging.Logger, d domain.Domain) error {
			app := server.Raw()
			app.Post(
				"/meter", func(ctx *fiber.Ctx) error {
					err := d.CreateMeter(ctx.Context(), &entities.Meter{
						Slug:          "test",
						Description:   "hello",
						EventType:     "test",
						Aggregation:   "sum",
						ValueProperty: "$.sample",
						GroupBy:       nil,
					})
					if errors.Is(err, domain.MeterAlreadyExistError) {
						return ctx.Status(http.StatusConflict).JSON(map[string]string{"status": "conflict"})
					}
					return err
				},
			)
			app.Delete(
				"/meter", func(ctx *fiber.Ctx) error {
					return ctx.Status(http.StatusAccepted).JSON(map[string]string{"status": "ok"})
				},
			)
			app.Post(
				"/event", func(ctx *fiber.Ctx) error {
					return ctx.Status(http.StatusAccepted).JSON(map[string]string{"status": "ok"})
				},
			)
			return nil
		},
	),
)
