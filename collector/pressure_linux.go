// Copyright 2019 The Prometheus Authors
// Portions Copyright 2021 Jens Elkner (jel+nex@cs.uni-magdeburg.de)
//
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

//go:build !nopressure
// +build !nopressure

package collector

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs"
)

var (
	psiResources = []string{"cpu", "io", "memory"}
)

type pressureStatsCollector struct {
	cpu     *prometheus.Desc
	io      *prometheus.Desc
	ioFull  *prometheus.Desc
	mem     *prometheus.Desc
	memFull *prometheus.Desc

	fs procfs.FS

	logger log.Logger
}

func init() {
	registerCollector("pressure", defaultEnabled, NewPressureStatsCollector)
}

// NewPressureStatsCollector returns a Collector exposing pressure stall information
func NewPressureStatsCollector(logger log.Logger) (Collector, error) {
	fs, err := procfs.NewFS(*procPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open procfs: %w", err)
	}

	return &pressureStatsCollector{
		cpu: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "psi", "cpu_some_us"),
			"Total share of time in µs in which at least some tasks are stalled on CPU time",
			nil, nil,
		),
		io: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "psi", "io_some_us"),
			"Total share of time in µs at least some tasks are stalled on IO",
			nil, nil,
		),
		ioFull: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "psi", "io_full_us"),
			"Total share of time in µs in which all non-idle tasks are stalled on IO simultaneously",
			nil, nil,
		),
		mem: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "psi", "memory_some_us"),
			"Total share of time in µs at least some tasks are stalled on memory",
			nil, nil,
		),
		memFull: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "psi", "memory_full_us"),
			"Total share of time in µs in which all non-idle tasks are stalled on memory simultaneously",
			nil, nil,
		),
		fs:     fs,
		logger: logger,
	}, nil
}

// Update calls procfs.NewPSIStatsForResource for the different resources and updates the values
func (c *pressureStatsCollector) Update(ch chan<- prometheus.Metric) error {
	for _, res := range psiResources {
		level.Debug(c.logger).Log("msg", "collecting statistics for resource", "resource", res)
		vals, err := c.fs.PSIStatsForResource(res)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				level.Debug(c.logger).Log("msg", "pressure information is unavailable, you need a Linux kernel >= 4.20 and/or CONFIG_PSI enabled for your kernel")
				return ErrNoData
			}
			if errors.Is(err, syscall.ENOTSUP) {
				level.Debug(c.logger).Log("msg", "pressure information is disabled, add psi=1 kernel command line to enable it")
				return ErrNoData
			}
			return fmt.Errorf("failed to retrieve pressure stats: %w", err)
		}
		switch res {
		case "cpu":
			ch <- prometheus.MustNewConstMetric(c.cpu, prometheus.CounterValue, float64(vals.Some))
		case "io":
			ch <- prometheus.MustNewConstMetric(c.io, prometheus.CounterValue, float64(vals.Some))
			ch <- prometheus.MustNewConstMetric(c.ioFull, prometheus.CounterValue, float64(vals.Full))
		case "memory":
			ch <- prometheus.MustNewConstMetric(c.mem, prometheus.CounterValue, float64(vals.Some))
			ch <- prometheus.MustNewConstMetric(c.memFull, prometheus.CounterValue, float64(vals.Full))
		default:
			level.Debug(c.logger).Log("msg", "did not account for resource", "resource", res)
		}
	}

	return nil
}
