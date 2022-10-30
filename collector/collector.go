// Copyright 2015 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package collector includes all individual collectors to gather and export system metrics.
package collector

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// Namespace defines the common namespace to be used by all metrics.
const Namespace = "node"

var (
	scrapeDurationDesc = prometheus.NewDesc(
		prometheus.BuildFQName(Namespace, "scrape", "collector_duration_seconds"),
		"node_exporter: Duration of a collector scrape.",
		[]string{"collector"},
		nil,
	)
	scrapeSuccessDesc = prometheus.NewDesc(
		prometheus.BuildFQName(Namespace, "scrape", "collector_success"),
		"node_exporter: Whether a collector succeeded.",
		[]string{"collector"},
		nil,
	)
)

const (
	defaultEnabled  = true
	defaultDisabled = false
)

var (
	factories              = make(map[string]func(lable string, logger log.Logger) (Collector, error))
	initiatedCollectorsMtx = sync.Mutex{}
	initiatedCollectors    = make(map[string]Collector)
	collectorState         = make(map[string]bool)
	forcedCollectors       = map[string]bool{} // collectors which have been explicitly enabled or disabled
)

func (n *NodeCollector) RegisterCollector(collector, lable string, factory func(lable string, logger log.Logger) (Collector, error)) {

	// flag := kingpin.Flag(flagName, flagHelp).Default(defaultValue).Action(collectorFlagAction(collector)).Bool()
	var new_key string = collector + lable

	fmt.Println("before collectorState :", collectorState)
	collectorState[new_key] = true
	fmt.Println("after collectorState :", collectorState)

	fmt.Println("before factories :", factories)
	factories[new_key] = factory
	fmt.Println("after factories :", factories)

	// f := make(map[string]bool)
	// for _, filter := range filters {
	// 	enabled, exist := collectorState[filter]
	// 	if !exist {
	// 		return nil, fmt.Errorf("missing collector: %s", filter)
	// 	}
	// 	if !enabled {
	// 		return nil, fmt.Errorf("disabled collector: %s", filter)
	// 	}
	// 	f[filter] = true
	// }

	collectors := make(map[string]Collector)
	initiatedCollectorsMtx.Lock()
	defer initiatedCollectorsMtx.Unlock()

	fmt.Printf("collectroState: %v\n", collectorState)
	for key, enabled := range collectorState {
		fmt.Println("in loop")
		// if !enabled || (len(f) > 0 && !f[key]) {
		// 	continue
		// }

		fmt.Println(key, enabled)
		collector, err := factories[key](lable, n.logger)
		if err != nil {
			return
		}
		collectors[key] = collector
		// initiatedCollectors[key] = collector
		fmt.Println("key:", key, "after map:", collectors)
	}
	fmt.Println("here", collectors)
	n.Collectors = collectors
}

// NodeCollector implements the prometheus.Collector interface.
type NodeCollector struct {
	Collectors map[string]Collector
	logger     log.Logger
}

// DisableDefaultCollectors sets the collector state to false for all collectors which
// have not been explicitly enabled on the command line.
func DisableDefaultCollectors() {
	fmt.Println(collectorState)

	for c := range collectorState {
		zap.S().Info(c)

		if _, ok := forcedCollectors[c]; !ok {
			collectorState[c] = false
		}
	}
}

// NewNodeCollector creates a new NodeCollector.
func NewNodeCollector(logger log.Logger) (*NodeCollector, error) {
	return &NodeCollector{
		Collectors: nil,
		logger:     logger,
	}, nil
}

// Describe implements the prometheus.Collector interface.
func (n NodeCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- scrapeDurationDesc
	ch <- scrapeSuccessDesc
}

// Collect implements the prometheus.Collector interface.
func (n NodeCollector) Collect(ch chan<- prometheus.Metric) {
	wg := sync.WaitGroup{}
	fmt.Println(n.Collectors)
	wg.Add(len(n.Collectors))
	for name, c := range n.Collectors {
		go func(name string, c Collector) {
			execute(name, c, ch, n.logger)
			wg.Done()
		}(name, c)
	}
	wg.Wait()
}

func execute(name string, c Collector, ch chan<- prometheus.Metric, logger log.Logger) {
	begin := time.Now()
	err := c.Update(ch)
	duration := time.Since(begin)
	var success float64

	if err != nil {
		if IsNoDataError(err) {
			level.Debug(logger).Log("msg", "collector returned no data", "name", name, "duration_seconds", duration.Seconds(), "err", err)
		} else {
			level.Error(logger).Log("msg", "collector failed", "name", name, "duration_seconds", duration.Seconds(), "err", err)
		}
		success = 0
	} else {
		level.Debug(logger).Log("msg", "collector succeeded", "name", name, "duration_seconds", duration.Seconds())
		success = 1
	}
	ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, duration.Seconds(), name)
	ch <- prometheus.MustNewConstMetric(scrapeSuccessDesc, prometheus.GaugeValue, success, name)
}

// Collector is the interface a collector has to implement.
type Collector interface {
	// Get new metrics and expose them via prometheus registry.
	Update(ch chan<- prometheus.Metric) error
}

// ErrNoData indicates the collector found no data to collect, but had no other error.
var ErrNoData = errors.New("collector returned no data")

func IsNoDataError(err error) bool {
	return err == ErrNoData
}
