package domain

import (
	"context"
	"crypto/md5"
	"fmt"
	"reflect"
	"time"

	"github.com/PaesslerAG/jsonpath"
	"github.com/kloudlite/kloudmeter/internal/domain/entities"
	"github.com/kloudlite/kloudmeter/pkg/kv"
	"github.com/kloudlite/kloudmeter/pkg/messaging/types"
	"github.com/nats-io/nats.go/jetstream"
)

type DContext struct {
	Context context.Context
	MsgTime time.Time
}

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

func (d *Impl) upsertReadings(ctx DContext, values upsertValues) error {
	reading, err := d.readingsRepo.Get(ctx.Context, values.key)
	if err != nil && err != jetstream.ErrKeyNotFound {
		return err
	}

	if err == jetstream.ErrKeyNotFound {
		return d.createReading(ctx, values)
	}

	return d.updateReading(ctx, reading, values)
}

func (d *Impl) updateReadings(ctx DContext, meter *entities.Meter, event *entities.Event) error {

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

			if err := d.meterProducer.Produce(ctx.Context, types.ProduceMsg{
				Subject: fmt.Sprintf("meters.event-errors.%s", event.Key()),
				Payload: b,
				MsgID:   &event.Id,
			}); err != nil {
				d.logger.Errorf(err, "failed to add produce message to dead letter queue")
			}
		}

	}

	for k, segment := range meter.GroupBy {

		val, err := dataOnPath[string](event.Data, segment)
		if err != nil {
			d.logger.Warnf("failed to get value for segment %s: %s", segment, err)
			continue
		}

		// md5 hash of the segment value
		hash := fmt.Sprintf("%x", md5.Sum([]byte(*val)))

		key := fmt.Sprintf("%s.%s.%s.%s", meter.Key(), event.Subject, k, hash)

		fmt.Println(key)
		if err := d.upsertReadings(ctx, upsertValues{
			meter:         meter,
			event:         event,
			segment:       k,
			key:           key,
			valueProperty: meter.ValueProperty,
		}); err != nil {
			d.logger.Errorf(err, "failed to updateReadings")

			b, err := event.ToJson()
			if err != nil {
				d.logger.Errorf(err, "faild to marshal json")
				continue
			}

			if err := d.meterProducer.Produce(ctx.Context, types.ProduceMsg{
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

func (d *Impl) updateReading(ctx DContext, reading *entities.Reading, values upsertValues) error {
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
		Unique:  reading.Unique,
	}

	switch values.meter.Aggregation {
	case entities.AggTypeCount:
		value.Count = reading.Count + 1
	case entities.AggTypeSum, entities.AggTypeAvg, entities.AggTypeMax, entities.AggTypeMin, entities.AggTypeDuration:
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

		case entities.AggTypeDuration:
			totalDurr := ctx.MsgTime.Sub(reading.DurationData.LastCalculated)
			totalUnit := totalDurr.Hours() * reading.DurationData.Unit
			value.DurationData = entities.DurationData{
				Total:          reading.DurationData.Total + totalUnit,
				Unit:           *val,
				LastCalculated: ctx.MsgTime,
			}
		}

		value.Count = reading.Count + 1
	case entities.AggTypeUnique:
		val, err := dataOnPath[string](values.event.Data, values.valueProperty)
		if err != nil {
			return err
		}

		value.Unique = reading.Unique

		if value.Unique == nil {
			value.Unique = make(map[string]int)
		}

		if _, ok := value.Unique[*val]; !ok {
			value.Unique[*val] = 1
		} else {
			value.Unique[*val] = value.Unique[*val] + 1
		}

	default:
		return fmt.Errorf("unknown aggregation type: %s", values.meter.Aggregation)
	}

	return d.readingsRepo.Set(ctx.Context, values.key, value)
}

func (d *Impl) createReading(ctx DContext, values upsertValues) error {

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
	case entities.AggTypeSum, entities.AggTypeAvg, entities.AggTypeMax, entities.AggTypeMin, entities.AggTypeDuration:
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
		case entities.AggTypeDuration:
			value.DurationData = entities.DurationData{
				Unit:           *val,
				LastCalculated: ctx.MsgTime,
				Total:          0,
			}
		}

	case entities.AggTypeUnique:
		val, err := dataOnPath[string](values.event.Data, values.valueProperty)
		if err != nil {
			return err
		}

		if value.Unique == nil {
			value.Unique = make(map[string]int)
		}

		if _, ok := value.Unique[*val]; !ok {
			value.Unique[*val] = 1
		} else {
			value.Unique[*val] = value.Unique[*val] + 1
		}

	default:
		return fmt.Errorf("unknown aggregation type: %s", values.meter.Aggregation)
	}

	return d.readingsRepo.Set(ctx.Context, values.key, value)
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
