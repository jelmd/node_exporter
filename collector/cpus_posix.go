// Copyright 2021 Jens Elkner (jel+prom@cs.uni-magdeburg.de)
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

//go:build !nocpus
// +build !nocpus

package collector

import (
	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
)

// #include <unistd.h>
import "C"						// requires .promu.yml::cgo: true

const metric = "cpus"

type cpusCollector struct {
	desc	*prometheus.Desc
	total	C.long
}

func init() {
	registerCollector(metric, defaultEnabled, NewCpusCollector)
}

func NewCpusCollector(logger log.Logger) (Collector, error) {
	return &cpusCollector{
		desc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, metric, "total"),
			"Total number of CPU cores or strands if HT or SMT is enabled.",
			// You need to restart node-exporter if the CPU configuration gets
			// changed.
			[]string{"state"}, nil,
		),
		total: 0,
	}, nil
}

func (c *cpusCollector) Update(ch chan<- prometheus.Metric) error {
	if c.total == 0 {
		// On linux it scans the /sys/devices/system/cpu/ for dirs starting
		// with 'cpu' - so relative expensive and run only once.
		// On Solaris a "cheap" syscall.
		c.total = C.sysconf(C._SC_NPROCESSORS_CONF)
	}
	// On linux this is a syscall now - counts the bits in the sched_affinity
	// mask - see also /sys/devices/system/cpu/online
	// On Solaris a "cheap" syscall.
	num := C.sysconf(C._SC_NPROCESSORS_ONLN)

	ch <- prometheus.MustNewConstMetric(
		c.desc, prometheus.GaugeValue, float64(num), "online",
	)

	ch <- prometheus.MustNewConstMetric(
		c.desc, prometheus.GaugeValue, float64(c.total - num), "offline",
	)
	return nil
}
