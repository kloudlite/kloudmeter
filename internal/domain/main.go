package domain

import (
	"context"
	"github.com/kloudlite/api/pkg/errors"
	"github.com/kloudlite/api/pkg/kv"
	"github.com/kloudlite/kloudmeter/internal/domain/entities"
	"github.com/kloudlite/kloudmeter/internal/env"
	"go.uber.org/fx"
)

type Impl struct {
	meterRepo kv.Repo[*entities.Meter]
}

func (d *Impl) CreateMeter(ctx context.Context, meter *entities.Meter) error {

	get, err := d.meterRepo.Get(ctx, meter.Slug)
	if err != nil && !d.meterRepo.ErrKeyNotFound(err) {
		return err
	}
	if get == nil {
		return d.meterRepo.Set(ctx, meter.Slug, meter)
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

func (d *Impl) RegisterEvent(ctx context.Context, event *entities.Event) (*entities.Event, error) {
	//TODO implement me
	panic("implement me")
}

var Module = fx.Module("domain", fx.Provide(func(e *env.Env,
	meterRepo kv.Repo[*entities.Meter]) (Domain, error) {
	return &Impl{
		meterRepo: meterRepo,
	}, nil
}))
