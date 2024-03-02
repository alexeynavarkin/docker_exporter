package stat

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/alexeynavarkin/docker_exporter/internal/metric"
	"github.com/alexeynavarkin/docker_exporter/internal/util"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/prometheus/client_golang/prometheus"
)

var labelsContainer = []string{"containerName", "serviceName", "serviceID", "type"}

type response struct {
	NetworkStats map[string]types.NetworkStats `json:"networks"`
	MemStats     types.MemoryStats             `json:"memory_stats"`
	CpuStats     types.CPUStats                `json:"cpu_stats"`
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
			labelsContainer,
		),
		metricMemStats: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "docker",
				Subsystem: "mem",
				Name:      "stat_bytes",
				Help:      "Container memory usage.",
			},
			labelsContainer,
		),
		metricNetStats: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "docker",
				Subsystem: "net",
				Name:      "stat_bytes",
				Help:      "Container network usage.",
			},
			labelsContainer,
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
			Filters: filters.NewArgs(filters.Arg("status", "running")),
		},
	)
	if err != nil {
		fmt.Printf("error get container list %v\n", err)
		return
	}

	var wg sync.WaitGroup
	wg.Add(len(containers))
	for _, c := range containers {
		fmt.Printf("collect stats for container %s\n", c.ID)
		go g.collectContainer(c, &wg)
	}
	wg.Wait()
	fmt.Print("collect done\n")
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

	containerName := strings.Join(c.Names, "")

	m, _ := g.metricMemStats.GetMetricWith(
		prometheus.Labels{
			"containerName": containerName,
			"serviceName":   util.GetMapValue(c.Labels, util.LabelNameServiceName, util.LabelDefaultValue),
			"serviceID":     util.GetMapValue(c.Labels, util.LabelNameServiceID, util.LabelDefaultValue),
			"type":          "used",
		},
	)
	m.Set(float64(resp.MemStats.Usage))

	m, _ = g.metricCpuStats.GetMetricWith(
		prometheus.Labels{
			"containerName": containerName,
			"serviceName":   util.GetMapValue(c.Labels, util.LabelNameServiceName, util.LabelDefaultValue),
			"serviceID":     util.GetMapValue(c.Labels, util.LabelNameServiceID, util.LabelDefaultValue),
			"type":          "usermode",
		},
	)
	m.Set(float64(resp.CpuStats.CPUUsage.UsageInUsermode))
	m, _ = g.metricCpuStats.GetMetricWith(
		prometheus.Labels{
			"containerName": containerName,
			"serviceName":   util.GetMapValue(c.Labels, util.LabelNameServiceName, util.LabelDefaultValue),
			"serviceID":     util.GetMapValue(c.Labels, util.LabelNameServiceID, util.LabelDefaultValue),
			"type":          "kernelmode",
		},
	)
	m.Set(float64(resp.CpuStats.CPUUsage.UsageInKernelmode))
	m, _ = g.metricCpuStats.GetMetricWith(
		prometheus.Labels{
			"containerName": containerName,
			"serviceName":   util.GetMapValue(c.Labels, util.LabelNameServiceName, util.LabelDefaultValue),
			"serviceID":     util.GetMapValue(c.Labels, util.LabelNameServiceID, util.LabelDefaultValue),
			"type":          "throttled",
		},
	)
	m.Set(float64(resp.CpuStats.ThrottlingData.ThrottledTime))

	for _, iface := range resp.NetworkStats {
		m, _ = g.metricNetStats.GetMetricWith(
			prometheus.Labels{
				"containerName": containerName,
				"serviceName":   util.GetMapValue(c.Labels, util.LabelNameServiceName, util.LabelDefaultValue),
				"serviceID":     util.GetMapValue(c.Labels, util.LabelNameServiceID, util.LabelDefaultValue),
				"type":          "rx",
			},
		)
		m.Add((float64(iface.RxBytes)))
		m, _ = g.metricNetStats.GetMetricWith(
			prometheus.Labels{
				"containerName": containerName,
				"serviceName":   util.GetMapValue(c.Labels, util.LabelNameServiceName, util.LabelDefaultValue),
				"serviceID":     util.GetMapValue(c.Labels, util.LabelNameServiceID, util.LabelDefaultValue),
				"type":          "tx",
			},
		)
		m.Add((float64(iface.TxBytes)))
		m, _ = g.metricNetStats.GetMetricWith(
			prometheus.Labels{
				"containerName": containerName,
				"serviceName":   util.GetMapValue(c.Labels, util.LabelNameServiceName, util.LabelDefaultValue),
				"serviceID":     util.GetMapValue(c.Labels, util.LabelNameServiceID, util.LabelDefaultValue),
				"type":          "drop",
			},
		)
		m.Add((float64(iface.RxDropped + iface.TxDropped)))
		m, _ = g.metricNetStats.GetMetricWith(
			prometheus.Labels{
				"containerName": containerName,
				"serviceName":   util.GetMapValue(c.Labels, util.LabelNameServiceName, util.LabelDefaultValue),
				"serviceID":     util.GetMapValue(c.Labels, util.LabelNameServiceID, util.LabelDefaultValue),
				"type":          "error",
			},
		)
		m.Add((float64(iface.RxErrors + iface.TxErrors)))
	}
}

func (g *Gatherer) Metrics() []prometheus.Collector {
	return []prometheus.Collector{
		g.metricCpuStats,
		g.metricMemStats,
		g.metricNetStats,
	}
}
