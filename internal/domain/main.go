package domain

import (
	"context"

	"github.com/kloudlite/kloudmeter/internal/domain/entities"
	"github.com/kloudlite/kloudmeter/internal/env"
	"github.com/kloudlite/kloudmeter/pkg/errors"
	"github.com/kloudlite/kloudmeter/pkg/kv"
	"github.com/kloudlite/kloudmeter/pkg/logging"
	"github.com/kloudlite/kloudmeter/pkg/nats"
	"go.uber.org/fx"
)

type Impl struct {
	meterRepo     kv.Repo[*entities.Meter]
	readingsRepo  kv.Repo[*entities.Reading]
	logger        logging.Logger
	meterMap      MeterMap
	oldMeterMap   MeterMap
	jc            *nats.JetstreamClient
	env           *env.Env
	meterProducer MeterProducer
}

func (d *Impl) ListMeters(ctx context.Context) ([]kv.Entry[*entities.Meter], error) {
	return d.meterRepo.Entries(ctx, ">")
}

func (d *Impl) RegisterMeter(ctx context.Context, meter entities.Meter) error {
	if err := meter.IsValid(); err != nil {
		return err
	}

	get, err := d.meterRepo.Get(ctx, meter.Key())
	if err != nil && !d.meterRepo.ErrKeyNotFound(err) {
		return err
	}

	if get == nil {
		return d.meterRepo.Set(ctx, meter.Key(), &meter)
	}

	return MeterAlreadyExistError
}

func (d *Impl) DeleteMeter(ctx context.Context, id string) error {
	return d.meterRepo.Drop(ctx, id)
}

func (d *Impl) GetMeter(ctx context.Context, id string) (*entities.Meter, error) {
	get, err := d.meterRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if get == nil {
		return nil, errors.New("meter not found")
	}
	return get, nil
}

var Module = fx.Module("domain", fx.Provide(func(e *env.Env,
	meterRepo kv.Repo[*entities.Meter],
	readingsRepo kv.Repo[*entities.Reading],
	logger logging.Logger,
	jc *nats.JetstreamClient,
	env *env.Env,
	meterProducer MeterProducer,
) (Domain, error) {
	return &Impl{
		meterRepo:     meterRepo,
		readingsRepo:  readingsRepo,
		logger:        logger,
		meterMap:      MeterMap{},
		oldMeterMap:   MeterMap{},
		jc:            jc,
		env:           env,
		meterProducer: meterProducer,
	}, nil
}))
