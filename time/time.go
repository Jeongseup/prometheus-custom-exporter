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

//go:build !notime
// +build !notime

package time

import (
	"fmt"
	"time"

	"example.com/collector"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

type typedDesc struct {
	desc      *prometheus.Desc
	valueType prometheus.ValueType
}

func (d *typedDesc) mustNewConstMetric(value float64, labels ...string) prometheus.Metric {
	return prometheus.MustNewConstMetric(d.desc, d.valueType, value, labels...)
}

type TimeCollector struct {
	now typedDesc
	// lable  string
	logger log.Logger
}

func init() {
	// zap.S().Info("here")
	fmt.Println("time register")
	// registerCollector("time", defaultEnabled, NewTimeCollector)
}

// NewTimeCollector returns a new Collector exposing the current system time in
// seconds since epoch.
func NewTimeCollector(lable string, logger log.Logger) (collector.Collector, error) {

	// deviceTempGaugeVector := prometheus.NewGaugeVec(
	// 	prometheus.GaugeOpts{
	// 		Name: "my_temperature_celsius",
	// 	},
	// 	[]string{
	// 		"device_id" // Using single label instead of 2 labels "id" and "value"
	// 	},
	// )
	// deviceTempGaugeVector.WithLabelValues()

	const subsystem = "time"
	return &TimeCollector{
		now: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName("", lable, "seconds"),
			"System time in seconds since epoch (1970).",
			nil, prometheus.Labels{lable: lable + "value"},
		), prometheus.GaugeValue},

		logger: logger,
	}, nil
}

func (c *TimeCollector) Update(ch chan<- prometheus.Metric) error {
	now := time.Now()
	nowSec := float64(now.UnixNano()) / 1e9
	// zone, zoneOffset := now.Zone()

	level.Debug(c.logger).Log("msg", "Return time", "now", nowSec)
	ch <- c.now.mustNewConstMetric(nowSec)

	// level.Debug(c.logger).Log("msg", "Zone offset", "offset", zoneOffset, "time_zone", c.lable)
	// ch <- c.zone.mustNewConstMetric(float64(zoneOffset), zone)

	return nil
}

// func (c *timeCollector) update(ch chan<- prometheus.Metric) error {
// 	return nil
// }
