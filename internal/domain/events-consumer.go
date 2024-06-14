package domain

import (
	"context"
	"fmt"
	"time"

	"github.com/kloudlite/kloudmeter/internal/domain/entities"
	"github.com/kloudlite/kloudmeter/pkg/errors"
	msg_nats "github.com/kloudlite/kloudmeter/pkg/messaging/nats"
	"github.com/kloudlite/kloudmeter/pkg/messaging/types"
	"github.com/nats-io/nats.go/jetstream"
)

type MeterMap map[string]*entities.Meter

func (d MeterMap) Add(meter *entities.Meter) {
	d[meter.Hash()] = meter
}

func (d MeterMap) AddMeters(meters []*entities.Meter) {
	for _, m := range meters {
		d.Add(m)
	}
}

func (d MeterMap) RemoveMeter(key string) {
	delete(d, key)
}
func (d *Impl) AddMeterToConsume(meter *entities.Meter) {
	d.meterMap.Add(meter)
}

func (d *Impl) RemoveMeterFromConsume(key string) {
	d.meterMap.RemoveMeter(key)
}

func (d *Impl) StartConsumingEvents(ctx context.Context) error {
	meters, err := d.meterRepo.List(ctx, ">")
	if err != nil && err != jetstream.ErrNoKeysFound {
		return errors.NewE(err)
	}

	d.meterMap.AddMeters(meters)

	upCh, downCh, runner := d.consumerController(ctx)
	go runner()

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		if err := func() error {
			for _, meter := range d.meterMap {
				if _, ok := d.oldMeterMap[meter.Hash()]; !ok {
					d.logger.Infof("new meter: (%s)(%s)", meter.Key(), meter.Hash())
					d.oldMeterMap.Add(meter)
					upCh <- meter
				}
			}

			for _, meter := range d.oldMeterMap {
				if _, ok := d.meterMap[meter.Hash()]; !ok {
					d.logger.Infof("removed meter: (%s)(%s)", meter.Key(), meter.Hash())
					d.oldMeterMap.RemoveMeter(meter.Hash())
					downCh <- meter
				}
			}

			return nil
		}(); err != nil {
			d.logger.Errorf(err, "error while consuming events")
		}

		time.Sleep(time.Second * 5)
	}
}

func (d *Impl) consumerController(ctx context.Context) (chan *entities.Meter, chan *entities.Meter, func()) {
	upCh := make(chan *entities.Meter)
	downCh := make(chan *entities.Meter)

	ctxMap := map[string]func(){}

	rnr := func() {
		for {
			select {
			case <-ctx.Done():
				return
			case up := <-upCh:
				go func() {
					if _, ok := ctxMap[up.Hash()]; ok {
						d.logger.Infof("consumer already running")
						return
					}

					consumerName := up.Hash()
					ctx, cf := context.WithCancel(context.TODO())
					ctxMap[consumerName] = cf

					consumer, err := msg_nats.NewJetstreamConsumer(ctx, d.jc, msg_nats.JetstreamConsumerArgs{
						Stream: d.env.MeterNatsStream,
						ConsumerConfig: msg_nats.ConsumerConfig{
							Name:        consumerName,
							Durable:     consumerName,
							Description: "this consumer reads message from a subject dedicated to errors, that occurred when the resource was applied at the agent",
							FilterSubjects: []string{
								fmt.Sprintf("meters.events.%s.>", up.EventType),
							},
						},
					})

					if err != nil {
						d.logger.Errorf(err, "error while creating consumer")
						return
					}

					if err := consumer.Consume(func(msg *types.ConsumeMsg) error {

						var event entities.Event
						if err := event.ParseBytes(msg.Payload); err != nil {
							return err
						}

						if err := d.updateReadings(ctx, up, &event); err != nil {
							d.logger.Errorf(err, "could not update readings")
						}

						return nil
					}, types.ConsumeOpts{
						OnError: func(err error) error {
							d.logger.Errorf(err, "error while consuming")
							return nil
						},
					}); err != nil {
						d.logger.Errorf(err, "error while consuming")
						return
					}

				}()

			case down := <-downCh:
				if f, ok := ctxMap[down.Hash()]; ok {
					f()
					delete(ctxMap, down.Hash())
				}
			}
		}
	}

	return upCh, downCh, rnr
}
