package events

import (
	"context"
	"hypervisor/internal/models"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

var Em *Emitter

type Config struct {
	Buffer     int
	BatchSize  int
	FlushEvery time.Duration
}

var (
	defaultConfig = Config{
		Buffer:     1000,
		BatchSize:  50,
		FlushEvery: 2 * time.Second,
	}
	fastConfig = Config{
		Buffer:     1000,
		BatchSize:  50,
		FlushEvery: 50 * time.Millisecond,
	}
)

type Emitter struct {
	coll       *mongo.Collection
	buf        chan models.Event
	cfg        Config
	deployment string

	wg        sync.WaitGroup
	onceClose sync.Once

	InsertOne  func(context.Context, models.Event) error
	InsertMany func(context.Context, []models.Event) error
}

func NewEmitter(coll *mongo.Collection, deployment string) *Emitter {
	return NewEmitterWithConfig(coll, deployment, selectConfig(deployment))
}

func NewEmitterWithConfig(coll *mongo.Collection, deployment string, cfg Config) *Emitter {
	e := &Emitter{
		coll:       coll,
		buf:        make(chan models.Event, cfg.Buffer),
		cfg:        cfg,
		deployment: deployment,
	}

	e.InsertOne = func(ctx context.Context, evt models.Event) error {
		_, err := e.coll.InsertOne(ctx, evt)
		return err
	}

	e.InsertMany = func(ctx context.Context, evts []models.Event) error {
		docs := make([]interface{}, len(evts))
		for i, evt := range evts {
			docs[i] = evt
		}

		_, err := e.coll.InsertMany(ctx, docs)
		return err
	}

	e.wg.Add(1)
	go e.worker()

	return e
}

func selectConfig(deployment string) Config {
	switch deployment {
	case "test":
		return fastConfig
	default:
		return defaultConfig
	}
}

func (e *Emitter) Close() {
	e.onceClose.Do(func() {
		close(e.buf)
		e.wg.Wait()
	})
}

func (e *Emitter) worker() {
	defer e.wg.Done()

	batch := make([]models.Event, 0, e.cfg.BatchSize)
	timer := time.NewTimer(e.cfg.FlushEvery)

	defer timer.Stop()

	flush := func() {
		if len(batch) == 0 {
			timer.Reset(e.cfg.FlushEvery)
			return
		}

		ctx, cancel := context.WithTimeout(
			context.Background(),
			2*time.Second,
		)

		_ = e.InsertMany(ctx, batch)

		cancel()

		batch = batch[:0]
		timer.Reset(e.cfg.FlushEvery)
	}

	for {
		select {
		case evt, ok := <-e.buf:
			if !ok {
				flush()
				return
			}

			batch = append(batch, evt)

			if len(batch) >= e.cfg.BatchSize {
				flush()
			}
		case <-timer.C:
			flush()
		}
	}
}
