package metric

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Gatherer interface {
	Metrics() []prometheus.Collector
	Gather()
}

type Collector struct {
	event *prometheus.CounterVec

	reg                *prometheus.Registry
	defaultHandlerFunc http.Handler
	gatherers          []Gatherer
}

func NewCollector() *Collector {
	reg := prometheus.NewRegistry()

	c := &Collector{
		event: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "docker",
				Name:      "event",
				Help:      "Docker events.",
			},
			[]string{"serviceName", "serviceID", "eventType"},
		),
		reg:                reg,
		defaultHandlerFunc: promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}),
	}

	reg.MustRegister(c.event)

	return c
}

func (c *Collector) RegisterGatherer(gatherer Gatherer) {
	for _, m := range gatherer.Metrics() {
		c.reg.MustRegister(m)
	}
	c.gatherers = append(c.gatherers, gatherer)
}

func (c *Collector) RegisterEvent(serviceName string, serviceID string, eventType string) {
	c.event.With(
		prometheus.Labels{
			"serviceName": serviceName,
			"serviceID":   serviceID,
			"eventType":   eventType,
		},
	).Inc()
}

func (c *Collector) handler() http.Handler {
	h := func(w http.ResponseWriter, r *http.Request) {
		for _, g := range c.gatherers {
			fmt.Printf("invoke gatherer\n")
			g.Gather()
		}
		fmt.Printf("invoke metric handler\n")
		c.defaultHandlerFunc.ServeHTTP(w, r)
	}
	return http.HandlerFunc(h)
}

func (c *Collector) ExposeHTTP(ctx context.Context) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", c.handler())

	s := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		err := s.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			fmt.Printf("server closed\n")
		} else if err != nil {
			fmt.Printf("error listen %s\n", err)
		}
	}()

	<-ctx.Done()
	err := s.Close()
	if err != nil {
		fmt.Printf("error close server %s\n", err)
	}
}
