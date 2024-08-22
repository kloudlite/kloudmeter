package app

import (
	"context"
	"crypto/md5"
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/kloudlite/kloudmeter/pkg/functions"
	httpServer "github.com/kloudlite/kloudmeter/pkg/http-server"
	"github.com/kloudlite/kloudmeter/pkg/logging"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/kloudlite/kloudmeter/internal/domain"
	"github.com/kloudlite/kloudmeter/internal/domain/entities"
	"github.com/kloudlite/kloudmeter/internal/env"

	"github.com/kloudlite/kloudmeter/pkg/kv"
	msg_nats "github.com/kloudlite/kloudmeter/pkg/messaging/nats"
	"github.com/kloudlite/kloudmeter/pkg/messaging/types"
	"github.com/kloudlite/kloudmeter/pkg/nats"

	"go.uber.org/fx"
)

// var TOPIC_NAME_PREFIX = "meters:event.*"

// type MeterConsumer messaging.Consumer

var Module = fx.Module("app",

	kv.NewNatsKvRepoFx[*entities.Meter]("meters"),
	kv.NewNatsKvRepoFx[*entities.Reading]("readings"),

	domain.Module,

	fx.Provide(func(jc *nats.JetstreamClient, ev *env.Env, logger logging.Logger) domain.MeterProducer {
		return msg_nats.NewJetstreamProducer(jc)
	}),

	fx.Invoke(func(lf fx.Lifecycle, d domain.Domain, logr logging.Logger) {
		lf.Append(fx.Hook{
			OnStart: func(context.Context) error {
				go func() {
					err := d.StartConsumingEvents(context.TODO())
					if err != nil {
						logr.Errorf(err, "could not process events")
					}
				}()
				return nil
			},
			OnStop: func(ctx context.Context) error {
				return nil
			},
		})
	}),

	fx.Invoke(
		func(server httpServer.Server, d domain.Domain, mp domain.MeterProducer) error {
			app := server.Raw()
			app.Post(
				"/api/create-meter", func(ctx *fiber.Ctx) error {

					var meter entities.Meter

					err := ctx.BodyParser(&meter)
					if err != nil {
						return ctx.Status(http.StatusBadRequest).JSON(map[string]string{"status": "error", "message": err.Error()})
					}

					if err := meter.IsValid(); err != nil {
						return ctx.Status(http.StatusBadRequest).JSON(map[string]string{"status": "error", "message": err.Error()})
					}

					if err = d.RegisterMeter(ctx.Context(), meter); err != nil {
						return ctx.Status(http.StatusBadRequest).JSON(map[string]string{"status": "error", "message": err.Error()})
					}

					d.AddMeterToConsume(&meter)

					return ctx.Status(http.StatusAccepted).JSON(map[string]string{"status": "ok"})
				},
			)

			app.Get(
				"/api/meters", func(ctx *fiber.Ctx) error {
					a, err := d.ListMeters(ctx.Context())
					if err != nil {
						return ctx.Status(http.StatusBadRequest).JSON(map[string]string{"status": "error", "message": err.Error()})
					}

					return ctx.Status(http.StatusOK).JSON(a)
				},
			)

			app.Get(
				"/api/reading", func(ctx *fiber.Ctx) error {
					key, err := d.GetReading(ctx.Context(), ctx.Query("key"))
					if err != nil {
						return ctx.Status(http.StatusBadRequest).JSON(map[string]string{"status": "error", "message": err.Error()})
					}

					return ctx.Status(http.StatusOK).JSON(key)
				},
			)

			app.Get(
				"/api/meter", func(ctx *fiber.Ctx) error {
					key, err := d.GetMeter(ctx.Context(), ctx.Query("key"))
					if err != nil {
						return ctx.Status(http.StatusBadRequest).JSON(map[string]string{"status": "error", "message": err.Error()})
					}

					return ctx.Status(http.StatusOK).JSON(key)
				},
			)

			app.Get("/api/readings", func(ctx *fiber.Ctx) error {
				a, err := d.ListReadings(ctx.Context(), ">")
				if err != nil {
					return ctx.Status(http.StatusBadRequest).JSON(map[string]string{"status": "error", "message": err.Error()})
				}

				return ctx.Status(http.StatusOK).JSON(a)
			})

			app.Delete(
				"/api/meter", func(ctx *fiber.Ctx) error {
					fmt.Println("delete meter", ctx.Query("key", ""))
					key := ctx.Query("key", "")

					if key == "" {
						return ctx.Status(http.StatusBadRequest).JSON(map[string]string{"status": "error", "message": "key is required"})
					}

					if err := d.DeleteMeter(ctx.Context(), key); err != nil {
						return ctx.Status(http.StatusBadRequest).JSON(map[string]string{"status": "error", "message": err.Error()})
					}

					d.RemoveMeterFromConsume(fmt.Sprintf("%x", md5.Sum([]byte(key))))

					return ctx.Status(http.StatusAccepted).JSON(map[string]string{"status": "ok"})
				},
			)

			app.Post(
				"/api/register-event", func(ctx *fiber.Ctx) error {
					var event entities.Event

					m, err := d.ListMeters(ctx.Context())
					if err != nil && err != jetstream.ErrKeyNotFound {
						return ctx.Status(http.StatusBadRequest).JSON(map[string]string{"status": "error", "message": err.Error()})
					}

					if len(m) == 0 {
						return ctx.Status(http.StatusBadRequest).JSON(map[string]string{"status": "error", "message": "no meter found with provided event type"})
					}

					if err := ctx.BodyParser(&event); err != nil {
						return ctx.Status(http.StatusBadRequest).JSON(map[string]string{"status": "error", "message": err.Error()})
					}

					if err := event.IsValid(); err != nil {
						return ctx.Status(http.StatusBadRequest).JSON(map[string]string{"status": "error", "message": err.Error()})
					}

					b, err := event.ToJson()
					if err != nil {
						return ctx.Status(http.StatusBadRequest).JSON(map[string]string{"status": "error", "message": err.Error()})
					}

					if err := mp.Produce(ctx.Context(), types.ProduceMsg{
						Subject: fmt.Sprintf("meters.events.%s", event.Key()),
						Payload: b,
						MsgID:   functions.New(event.Id),
					}); err != nil {
						return ctx.Status(http.StatusBadRequest).JSON(map[string]string{"status": "error", "message": err.Error()})
					}

					return ctx.Status(http.StatusAccepted).JSON(map[string]string{"status": "ok"})
				},
			)

			app.Get("/healthy", func(ctx *fiber.Ctx) error {
				return ctx.Status(http.StatusOK).Send([]byte("OK"))
			})

			app.All("*", func(ctx *fiber.Ctx) error {
				return ctx.Status(http.StatusNotFound).JSON(map[string]string{"status": "error", "message": "not found"})
			})

			return nil
		},
	),
)
