package domain

import (
	"context"
	"github.com/kloudlite/api/pkg/errors"
	"github.com/kloudlite/kloudmeter/internal/domain/entities"
)

var MeterAlreadyExistError = errors.New("meter already exist")

type Domain interface {
	CreateMeter(ctx context.Context, meter *entities.Meter) error
	DeleteMeter(ctx context.Context, id string) error
	GetMeter(ctx context.Context, id string) (*entities.Meter, error)
	RegisterEvent(ctx context.Context, event *entities.Event) (*entities.Event, error)
}
