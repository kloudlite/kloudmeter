package nats

import (
	"context"
	"github.com/kloudlite/kloudmeter/pkg/errors"

	"github.com/kloudlite/kloudmeter/pkg/logging"
	"github.com/nats-io/nats.go/jetstream"
)

type JetstreamClient struct {
	Jetstream jetstream.JetStream
	Logger    logging.Logger
}

type ConsumerManager interface {
	GetConsumerInfo(ctx context.Context, stream string, consumer string) (*jetstream.ConsumerInfo, error)
	ListConsumers(ctx context.Context, stream string) ([]*jetstream.ConsumerInfo, error)
	DeleteConsumer(ctx context.Context, stream string, consumer string) error
	CreateStream(ctx context.Context, streamConfig *jetstream.StreamConfig) (*jetstream.Stream, error)
}

var _ ConsumerManager = (*JetstreamClient)(nil)

// DeleteConsumer implements ConsumerManager.
func (jc *JetstreamClient) DeleteConsumer(ctx context.Context, stream string, consumer string) error {
	err := jc.Jetstream.DeleteConsumer(ctx, stream, consumer)
	return errors.NewE(err)
}

// ListConsumers implements ConsumerManager.
func (jc *JetstreamClient) ListConsumers(ctx context.Context, stream string) ([]*jetstream.ConsumerInfo, error) {
	s, err := jc.Jetstream.Stream(ctx, stream)
	if err != nil {
		return nil, errors.NewE(err)
	}

	consumers := make([]*jetstream.ConsumerInfo, 0, 5)

	cil := s.ListConsumers(ctx)
	for ci := range cil.Info() {
		consumers = append(consumers, ci)
	}

	return consumers, nil
}

// GetConsumerInfo implements ConsumerManager
func (jc *JetstreamClient) GetConsumerInfo(ctx context.Context, stream string, consumer string) (*jetstream.ConsumerInfo, error) {
	s, err := jc.Jetstream.Stream(ctx, stream)
	if err != nil {
		return nil, errors.NewE(err)
	}

	c, err := s.Consumer(ctx, consumer)
	if err != nil {
		return nil, errors.NewE(err)
	}

	return c.Info(ctx)
}

func (jc *JetstreamClient) CreateStream(ctx context.Context, streamConfig *jetstream.StreamConfig) (*jetstream.Stream, error) {
	stream, err := jc.Jetstream.Stream(ctx, streamConfig.Name)
	if err == nil {
		return &stream, nil
	}

	stream, err = jc.Jetstream.CreateStream(ctx, *streamConfig)
	if err != nil {
		return nil, errors.NewE(err)
	}

	return &stream, nil
}

func NewJetstreamClient(nc *Client) (*JetstreamClient, error) {
	js, err := jetstream.New(nc.Conn)
	if err != nil {
		return nil, errors.NewE(err)
	}

	return &JetstreamClient{
		Jetstream: js,
		Logger:    nc.logger,
	}, nil
}
