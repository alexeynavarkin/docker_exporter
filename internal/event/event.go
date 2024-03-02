package event

import (
	"context"
	"fmt"
	"time"

	"github.com/alexeynavarkin/docker_exporter/internal/metric"
	"github.com/alexeynavarkin/docker_exporter/internal/util"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
)

const (
	ActionTypeOOM = "oom"
)

type Handler struct {
	cli             *client.Client
	metricCollector *metric.Collector
}

func NewHandler(cli *client.Client, metricCollector *metric.Collector) *Handler {
	return &Handler{
		cli:             cli,
		metricCollector: metricCollector,
	}
}

func (h *Handler) HandleEvents(ctx context.Context) {
	for {
		runCtx, runCancel := context.WithCancel(ctx)
		go h.handle(ctx, runCancel)
		<-runCtx.Done()

		select {
		case <-ctx.Done():
			return
		default:
		}

		fmt.Printf("will try to reconnect after 1 sec\n")
		time.Sleep(time.Second)
	}
}

func (h *Handler) handle(ctx context.Context, cancel func()) {
	defer cancel()

	fmt.Printf("start listen events\n")
	msgCh, errCh := h.cli.Events(ctx, types.EventsOptions{})
	for {
		select {
		case ev := <-msgCh:
			fmt.Printf("captured event: %+v\n", ev)
			h.handleEvent(ev)
		case ev := <-errCh:
			fmt.Printf("captured error event %v\n", ev)
			return
		case <-ctx.Done():
			fmt.Printf("captured stop event, shut down\n")
			return
		}
	}
}

func (h *Handler) handleEvent(ev events.Message) {
	switch ev.Type {
	case events.ContainerEventType:
		switch ev.Action {
		case ActionTypeOOM:
			h.metricCollector.RegisterEvent(
				ev.Actor.Attributes["name"],
				util.GetMapValue(ev.Actor.Attributes, util.LabelNameServiceName, util.LabelDefaultValue),
				util.GetMapValue(ev.Actor.Attributes, util.LabelNameServiceID, util.LabelDefaultValue),
				ev.Action,
			)
		}
	}
}
