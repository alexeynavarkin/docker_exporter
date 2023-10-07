package stat

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/alexeynavarkin/docker_exporter/internal/metric"
	"github.com/alexeynavarkin/docker_exporter/internal/util"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/prometheus/client_golang/prometheus"
)

type response struct {
	CpuStats     types.CPUStats                `json:"cpu_stats"`
	MemStats     types.MemoryStats             `json:"memory_stats"`
	NetworkStats map[string]types.NetworkStats `json:"networks"`
}

type Gatherer struct {
	metricCpuStats *prometheus.GaugeVec
	metricMemStats *prometheus.GaugeVec
	metricNetStats *prometheus.GaugeVec

	cli             *client.Client
	metricCollector *metric.Collector
}

func NewGatherer(cli *client.Client, metricCollector *metric.Collector) *Gatherer {
	return &Gatherer{
		metricCpuStats: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "docker",
				Subsystem: "cpu",
				Name:      "stat_ns",
				Help:      "Container cpu usage.",
			},
			[]string{"serviceName", "serviceID", "type"},
		),
		metricMemStats: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "docker",
				Subsystem: "mem",
				Name:      "stat_bytes",
				Help:      "Container memory usage.",
			},
			[]string{"serviceName", "serviceID", "type"},
		),
		metricNetStats: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "docker",
				Subsystem: "net",
				Name:      "stat_bytes",
				Help:      "Container network usage.",
			},
			[]string{"serviceName", "serviceID", "type"},
		),
		cli:             cli,
		metricCollector: metricCollector,
	}
}

func (g *Gatherer) Gather() {
	g.metricCpuStats.Reset()
	g.metricMemStats.Reset()
	g.metricNetStats.Reset()

	containers, err := g.cli.ContainerList(
		context.Background(),
		types.ContainerListOptions{
			Filters: filters.NewArgs(filters.Arg("status", "running"))},
	)
	if err != nil {
		fmt.Printf("error get container list %v\n", err)
		return
	}

	var wg sync.WaitGroup
	wg.Add(len(containers))
	for _, c := range containers {
		fmt.Printf("container %+v\n", c)
		go g.collectContainer(c, &wg)
	}
	wg.Wait()
}

func (g *Gatherer) collectContainer(c types.Container, wg *sync.WaitGroup) {
	defer wg.Done()

	stats, _ := g.cli.ContainerStats(context.Background(), c.ID, false)
	data, _ := io.ReadAll(stats.Body)
	var resp response
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return
	}

	m, _ := g.metricMemStats.GetMetricWith(
		prometheus.Labels{
			"serviceName": util.GetMapValue(c.Labels, util.LabelNameServiceName, util.LabelDefaultValue),
			"serviceID":   util.GetMapValue(c.Labels, util.LabelNameServiceID, util.LabelDefaultValue),
			"type":        "used",
		},
	)
	m.Set(float64(resp.MemStats.Usage))

	fmt.Printf("response %+v\n", resp)
}

func (g *Gatherer) Metrics() []prometheus.Collector {
	return []prometheus.Collector{
		g.metricCpuStats,
		g.metricMemStats,
		g.metricNetStats,
	}
}
