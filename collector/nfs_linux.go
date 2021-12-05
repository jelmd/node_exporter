// Copyright 2016 The Prometheus Authors
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

//go:build !nonfs
// +build !nonfs

package collector

import (
	"errors"
	"fmt"
	"os"
	"reflect"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs/nfs"
)

const (
	nfsSubsystem = "nfs"
)

type nfsCollector struct {
	fs               nfs.FS
	nfsRpcOpDesc     *prometheus.Desc
	nfsV2callDesc    *prometheus.Desc
	nfsV3callDesc    *prometheus.Desc
	nfsV4callDesc    *prometheus.Desc
	logger           log.Logger
}

func init() {
	registerCollector("nfs", defaultEnabled, NewNfsCollector)
}

// NewNfsCollector returns a new Collector exposing NFS statistics.
func NewNfsCollector(logger log.Logger) (Collector, error) {
	fs, err := nfs.NewFS(*procPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open procfs: %w", err)
	}

	return &nfsCollector{
		fs: fs,
		nfsRpcOpDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, nfsSubsystem, "rpc_ops"),
			"Total number of RPC operations made by the NFS client.",
			[]string{"name"}, nil,
		),
		nfsV2callDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, nfsSubsystem, "v2_calls"),
			"Number of NFS v2 calls made by the client.",
			[]string{"name"}, nil,
		),
		nfsV3callDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, nfsSubsystem, "v3_calls"),
			"Number of NFS v3 calls made by the client.",
			[]string{"name"}, nil,
		),
		nfsV4callDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, nfsSubsystem, "v4_calls"),
			"Number of NFS v4 calls made by the client.",
			[]string{"name"}, nil,
		),
		logger: logger,
	}, nil
}

func (c *nfsCollector) Update(ch chan<- prometheus.Metric) error {
	stats, err := c.fs.ProcNetRpcNfsStats()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			level.Debug(c.logger).Log("msg", "Not collecting NFS metrics", "err", err)
			return ErrNoData
		}
		return fmt.Errorf("failed to retrieve nfs stats: %w", err)
	}

	c.updateNFSClientRPCStats(ch, &stats.RpcClient)
	c.updateNFSRequestsv2Stats(ch, &stats.V2stats)
	c.updateNFSRequestsv3Stats(ch, &stats.V3stats)
	c.updateNFSRequestsv4Stats(ch, &stats.V4statsClient)

	return nil
}

// updateNFSClientRPCStats collects statistics for kernel server RPCs.
func (c *nfsCollector) updateNFSClientRPCStats(ch chan<- prometheus.Metric, s *nfs.RpcClient) {
	ch <- prometheus.MustNewConstMetric(c.nfsRpcOpDesc, prometheus.CounterValue, float64(s.RPCCount), "request")
	ch <- prometheus.MustNewConstMetric(c.nfsRpcOpDesc, prometheus.CounterValue, float64(s.Retransmissions), "retransmit")
	ch <- prometheus.MustNewConstMetric(c.nfsRpcOpDesc, prometheus.CounterValue, float64(s.AuthRefreshes), "authrefresh")
}

// updateNFSRequestsv2Stats collects statistics for NFSv2 requests.
func (c *nfsCollector) updateNFSRequestsv2Stats(ch chan<- prometheus.Metric, s *nfs.V2stats) {
	v := reflect.ValueOf(s).Elem()
	for i := int(s.Fields); i > 0; i-- {
		field := v.Field(i)
		ch <- prometheus.MustNewConstMetric(c.nfsV2callDesc, prometheus.CounterValue, float64(field.Uint()), v.Type().Field(i).Name)
	}
}

// updateNFSRequestsv3Stats collects statistics for NFSv3 requests.
func (c *nfsCollector) updateNFSRequestsv3Stats(ch chan<- prometheus.Metric, s *nfs.V3stats) {
	v := reflect.ValueOf(s).Elem()
	for i := int(s.Fields); i > 0; i-- {
		field := v.Field(i)
		ch <- prometheus.MustNewConstMetric(c.nfsV3callDesc, prometheus.CounterValue, float64(field.Uint()), v.Type().Field(i).Name)
	}
}

// updateNFSRequestsv4Stats collects statistics for NFSv4 requests.
func (c *nfsCollector) updateNFSRequestsv4Stats(ch chan<- prometheus.Metric, s *nfs.V4statsClient) {
	v := reflect.ValueOf(s).Elem()
	for i := int(s.Fields); i > 0; i-- {
		field := v.Field(i)
		ch <- prometheus.MustNewConstMetric(c.nfsV4callDesc, prometheus.CounterValue, float64(field.Uint()), v.Type().Field(i).Name)
	}
}
