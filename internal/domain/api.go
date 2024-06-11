package domain

import (
	"context"

	"github.com/kloudlite/kloudmeter/internal/domain/entities"
	"github.com/kloudlite/kloudmeter/pkg/errors"
	"github.com/kloudlite/kloudmeter/pkg/kv"
	"github.com/kloudlite/kloudmeter/pkg/messaging"
)

var MeterAlreadyExistError = errors.New("meter already exist")

type MeterProducer messaging.Producer

type Domain interface {
	RegisterMeter(ctx context.Context, meter entities.Meter) error
	ListMeters(ctx context.Context) ([]kv.Entry[*entities.Meter], error)
	DeleteMeter(ctx context.Context, id string) error
	GetMeter(ctx context.Context, id string) (*entities.Meter, error)

	ListReadings(ctx context.Context, pattern string) ([]kv.Entry[*entities.Reading], error)

	StartConsumingEvents(ctx context.Context) error

	AddMeterToConsume(meter *entities.Meter)
	RemoveMeterFromConsume(key string)
}
