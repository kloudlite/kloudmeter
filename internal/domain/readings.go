package domain

import (
	"context"
	"fmt"
	"reflect"

	"github.com/PaesslerAG/jsonpath"
	"github.com/kloudlite/kloudmeter/internal/domain/entities"
	"github.com/kloudlite/kloudmeter/pkg/kv"
	"github.com/kloudlite/kloudmeter/pkg/messaging/types"
	"github.com/nats-io/nats.go/jetstream"
)

func (d *Impl) ListReadings(ctx context.Context, pattern string) ([]kv.Entry[*entities.Reading], error) {
	return d.readingsRepo.Entries(ctx, pattern)
}

type upsertValues struct {
	meter         *entities.Meter
	event         *entities.Event
	segment       string
	key           string
	valueProperty string
}

func (d *Impl) upsertReadings(ctx context.Context, values upsertValues) error {
	reading, err := d.readingsRepo.Get(ctx, values.key)
	if err != nil && err != jetstream.ErrKeyNotFound {
		return err
	}

	if err == jetstream.ErrKeyNotFound {
		return d.createReading(ctx, values)
	}

	return d.updateReading(ctx, reading, values)
}

func (d *Impl) updateReadings(ctx context.Context, meter *entities.Meter, event *entities.Event) error {

	key := fmt.Sprintf("%s.%s", meter.Key(), event.Subject)
	if err := d.upsertReadings(ctx, upsertValues{
		meter:         meter,
		event:         event,
		segment:       "",
		key:           key,
		valueProperty: meter.ValueProperty,
	}); err != nil {
		d.logger.Errorf(err, "failed to updateReadings")

		b, err := event.ToJson()
		if err != nil {
			d.logger.Errorf(err, "faild to marshal json")
		} else {

			if err := d.meterProducer.Produce(ctx, types.ProduceMsg{
				Subject: fmt.Sprintf("meters.event-errors.%s", event.Key()),
				Payload: b,
				MsgID:   &event.Id,
			}); err != nil {
				d.logger.Errorf(err, "failed to add produce message to dead letter queue")
			}
		}

	}

	for k, segment := range meter.GroupBy {
		key := fmt.Sprintf("%s.%s.%s", meter.Key(), event.Subject, k)
		if err := d.upsertReadings(ctx, upsertValues{
			meter:         meter,
			event:         event,
			segment:       k,
			key:           key,
			valueProperty: segment,
		}); err != nil {
			d.logger.Errorf(err, "failed to updateReadings")

			b, err := event.ToJson()
			if err != nil {
				d.logger.Errorf(err, "faild to marshal json")
				continue
			}

			if err := d.meterProducer.Produce(ctx, types.ProduceMsg{
				Subject: fmt.Sprintf("meters.event-errors.%s", event.Key()),
				Payload: b,
				MsgID:   &event.Id,
			}); err != nil {
				d.logger.Errorf(err, "failed to add produce message to dead letter queue")
			}

		}
	}

	return nil
}

func (d *Impl) updateReading(ctx context.Context, reading *entities.Reading, values upsertValues) error {
	value := &entities.Reading{
		Event:   reading.Event,
		MeterId: reading.MeterId,
		Subject: reading.Subject,
		Segment: reading.Segment,
		Type:    reading.Type,
		Sum:     reading.Sum,
		Avg:     reading.Avg,
		Max:     reading.Max,
		Min:     reading.Min,
		Count:   reading.Count,
	}

	switch values.meter.Aggregation {
	case entities.AggTypeCount:
		value.Count = reading.Count + 1
	case entities.AggTypeSum, entities.AggTypeAvg, entities.AggTypeMax, entities.AggTypeMin:
		val, err := dataOnPath[float64](values.event.Data, values.valueProperty)
		if err != nil {
			return err
		}

		switch values.meter.Aggregation {
		case entities.AggTypeSum:
			value.Sum = reading.Sum + *val

		case entities.AggTypeAvg:
			value.Avg = ((reading.Avg * float64(reading.Count)) + *val) / float64((reading.Count + 1))

		case entities.AggTypeMax:
			if value.Max < *val {
				value.Max = *val
			}
		case entities.AggTypeMin:
			if value.Min > *val {
				value.Min = *val
			}
		}

		value.Count = reading.Count + 1
	default:
		return fmt.Errorf("unknown aggregation type: %s", values.meter.Aggregation)
	}

	return d.readingsRepo.Set(ctx, values.key, value)
}

func (d *Impl) createReading(ctx context.Context, values upsertValues) error {

	value := &entities.Reading{
		Event:   values.meter.EventType,
		MeterId: values.meter.Id,
		Segment: values.segment,
		Subject: values.event.Subject,
		Type:    values.meter.Aggregation,
		Count:   1,
	}

	switch values.meter.Aggregation {

	case entities.AggTypeCount:
	case entities.AggTypeSum, entities.AggTypeAvg, entities.AggTypeMax, entities.AggTypeMin:
		val, err := dataOnPath[float64](values.event.Data, values.valueProperty)
		if err != nil {
			return err
		}

		switch values.meter.Aggregation {
		case entities.AggTypeSum:
			value.Sum = *val

		case entities.AggTypeAvg:
			value.Avg = *val

		case entities.AggTypeMax:
			value.Max = *val

		case entities.AggTypeMin:
			value.Min = *val
		}

	default:
		return fmt.Errorf("unknown aggregation type: %s", values.meter.Aggregation)
	}

	return d.readingsRepo.Set(ctx, values.key, value)
}

func dataOnPath[T any](data map[string]any, jsPath string) (*T, error) {

	i, err := jsonpath.Get(jsPath, data)
	if err != nil {
		return nil, err
	}

	// Check if the type of i matches the type of T
	value := reflect.ValueOf(i)
	targetType := reflect.TypeOf((*T)(nil)).Elem()

	if value.Type().ConvertibleTo(targetType) {
		convertedValue := value.Convert(targetType).Interface().(T)
		return &convertedValue, nil
	}

	return nil, fmt.Errorf("type mismatch: expected %s but got %T (%v)", targetType, i, i)
}
