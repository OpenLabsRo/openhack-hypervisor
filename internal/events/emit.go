package events

import (
	"context"
	"hypervisor/internal/models"
	"time"
)

const (
	ActorHyperUser = "hyperuser"
	ActorSystem    = "system"
)

const (
	TargetHyperUser = "hyperuser"
)

func (e *Emitter) Emit(evt models.Event) {
	loc, err := time.LoadLocation("Europe/Bucharest")
	if err != nil {
		panic(err)
	}

	evt.TimeStamp = time.Now().In(loc)

	select {
	case e.buf <- evt:
	default:
		ctx, cancel := context.WithTimeout(
			context.Background(),
			2*time.Second,
		)
		defer cancel()

		_ = e.InsertOne(ctx, evt)
	}
}
