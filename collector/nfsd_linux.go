// Copyright 2018 The Prometheus Authors
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

//go:build !nonfsd
// +build !nonfsd

package collector

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs/nfs"

	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	skipProto = kingpin.Flag("collector.nfsd.skip", "Skip stats for the given comma separated list of NFS versions, i.e. 2, 3, 4, or 4ops.").Default("").String()
)

// A nfsdCollector is a Collector which gathers metrics from /proc/net/rpc/nfsd.
type nfsdCollector struct {
	fs                nfs.FS
	replyCacheDesc   *prometheus.Desc
	fhStaleDesc      *prometheus.Desc
	ioDesc           *prometheus.Desc
	thDesc           *prometheus.Desc
	rpcMsgDesc       *prometheus.Desc
	rpcTcpConnDesc   *prometheus.Desc
	rpcCallCheckDesc *prometheus.Desc
	nfsV2callDesc    *prometheus.Desc
	nfsV3callDesc    *prometheus.Desc
	nfsV4callDesc    *prometheus.Desc
	nfsV4opDesc      *prometheus.Desc
	skipV2           bool
	skipV3           bool
	skipV4           bool
	skipV4ops        bool
	logger           log.Logger
}

func init() {
	registerCollector("nfsd", defaultEnabled, NewNFSdCollector)
}

const (
	nfsdSubsystem = "nfsd"
)

// NewNFSdCollector returns a new Collector exposing /proc/net/rpc/nfsd stats.
func NewNFSdCollector(logger log.Logger) (Collector, error) {
	fs, err := nfs.NewFS(*procPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open procfs: %w", err)
	}
	skipV2, skipV3, skipV4, skipV4ops := false, false, false, false
	v := strings.Split(*skipProto,",")
	for _, s := range v {
		s = strings.TrimSpace(s)
		if s == "2" {
			skipV2 = true;
		} else if s == "3" {
			skipV3 = true;
		} else if s == "4" {
			skipV4 = true;
		} else if s == "4ops" {
			skipV4ops = true;
		} else {
			level.Warn(logger).Log("msg", "Unknown NFS version", s , "ignored.")
		}
	}

	return &nfsdCollector{
		fs: fs,
		replyCacheDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, nfsdSubsystem, "reply_cache_ops"),
			"Reply Cache operations by result.",
			[]string{"name"}, nil,
		),
		fhStaleDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, nfsdSubsystem, "file_handles"),
			"Total number of file handles by type.",
			[]string{"type"}, nil,
		),
		ioDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, nfsdSubsystem, "io_bytes"),
			"Total number of bytes returned to read or passed in write requests.",
			[]string{"op"}, nil,
		),
		thDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, nfsdSubsystem, "threads"),
			"Total number of configured NFSd kernel threads.",
			nil, nil,
		),
		rpcMsgDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, nfsdSubsystem, "rpc_messages"),
			"Total number of RPC messages received by protocol.",
			[]string{"proto"}, nil,
		),
		rpcTcpConnDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, nfsdSubsystem, "tcp_connections"),
			"Total number of accepted TCP connections.",
			nil, nil,
		),
		rpcCallCheckDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, nfsdSubsystem, "rpc_checks"),
			"Total number of RPCs received by syntactic check result.",
			[]string{"res"}, nil,
		),
		nfsV2callDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, nfsdSubsystem, "v2_calls"),
			"Total number of received NFS v2 calls by name.",
			[]string{"name"}, nil,
		),
		nfsV3callDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, nfsdSubsystem, "v3_calls"),
			"Total number of received NFS v3 calls by name.",
			[]string{"name"}, nil,
		),
		nfsV4callDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, nfsdSubsystem, "v4_calls"),
			"Total number of received NFS v4 calls by name.",
			[]string{"name"}, nil,
		),
		nfsV4opDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, nfsdSubsystem, "v4_ops"),
			"Total number of NFS v4 operations by name.",
			[]string{"name"}, nil,
		),
		skipV2: skipV2,
		skipV3: skipV3,
		skipV4: skipV4,
		skipV4ops: skipV4ops,
		logger: logger,
	}, nil
}

// Update implements Collector.
func (c *nfsdCollector) Update(ch chan<- prometheus.Metric) error {
	stats, err := c.fs.ProcNetRpcNfsdStats()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			level.Debug(c.logger).Log("msg", "Not collecting NFSd metrics", "err", err)
			return ErrNoData
		}
		return fmt.Errorf("failed to retrieve nfsd stats: %w", err)
	}

	c.updateNFSdReplyCacheStats(ch, &stats.ReplyCache)
	c.updateNFSdFileHandlesStats(ch, &stats.FileHandles)
	c.updateNFSdInputOutputStats(ch, &stats.InputOutput)
	c.updateNFSdThreadsStats(ch, &stats.Threads)
	c.updateNFSdNetworkStats(ch, &stats.Network)
	c.updateNFSdServerRPCStats(ch, &stats.RpcServer)
	c.updateNFSdRequestsV2Stats(ch, &stats.V2stats)
	c.updateNFSdRequestsV3Stats(ch, &stats.V3stats)
	c.updateNFSdRequestsV4Stats(ch, &stats.V4statsServer)
	c.updateNFSdRequestsV4Ops(ch, &stats.V4ops)

	return nil
}

// updateNFSdReplyCacheStats collects statistics for the reply cache.
func (c *nfsdCollector) updateNFSdReplyCacheStats(ch chan<- prometheus.Metric, s *nfs.ReplyCache) {
	ch <- prometheus.MustNewConstMetric(c.replyCacheDesc, prometheus.CounterValue, float64(s.Hits), "hit")
	ch <- prometheus.MustNewConstMetric(c.replyCacheDesc, prometheus.CounterValue, float64(s.Misses), "miss")
	ch <- prometheus.MustNewConstMetric(c.replyCacheDesc, prometheus.CounterValue, float64(s.NoCache), "nocache")
}

// updateNFSdFileHandlesStats collects statistics for the file handles.
func (c *nfsdCollector) updateNFSdFileHandlesStats(ch chan<- prometheus.Metric, s *nfs.FileHandles) {
	ch <- prometheus.MustNewConstMetric(c.fhStaleDesc, prometheus.CounterValue, float64(s.Stale), "stale")
	// NOTE: All other values are always 0
}

// updateNFSdInputOutputStats collects statistics for the bytes in/out.
func (c *nfsdCollector) updateNFSdInputOutputStats(ch chan<- prometheus.Metric, s *nfs.InputOutput) {
	ch <- prometheus.MustNewConstMetric(c.ioDesc, prometheus.CounterValue, float64(s.Read), "read")
	ch <- prometheus.MustNewConstMetric(c.ioDesc, prometheus.CounterValue, float64(s.Write), "write")
}

// updateNFSdThreadsStats collects statistics for kernel server threads.
func (c *nfsdCollector) updateNFSdThreadsStats(ch chan<- prometheus.Metric, s *nfs.Threads) {
	ch <- prometheus.MustNewConstMetric(c.thDesc, prometheus.GaugeValue, float64(s.Threads))
	// NOTE: all other values are always 0 since 2.6.32 (scalability impact)
}

// updateNFSdNetworkStats collects statistics for network packets/connections.
func (c *nfsdCollector) updateNFSdNetworkStats(ch chan<- prometheus.Metric, s *nfs.Network) {
	ch <- prometheus.MustNewConstMetric(c.rpcMsgDesc, prometheus.CounterValue, float64(s.NetCount), "any")
	ch <- prometheus.MustNewConstMetric(c.rpcMsgDesc, prometheus.CounterValue, float64(s.UDPCount), "udp")
	ch <- prometheus.MustNewConstMetric(c.rpcMsgDesc, prometheus.CounterValue, float64(s.TCPCount), "tcp")
	ch <- prometheus.MustNewConstMetric(c.rpcTcpConnDesc, prometheus.CounterValue, float64(s.TCPConnect))
}

// updateNFSdServerRPCStats collects statistics for kernel server RPCs.
func (c *nfsdCollector) updateNFSdServerRPCStats(ch chan<- prometheus.Metric, s *nfs.RpcServer) {
	ch <- prometheus.MustNewConstMetric(c.rpcCallCheckDesc, prometheus.CounterValue, float64(s.Good), "good")
	// skip s.Bad because this is the sum of bad_*
	ch <- prometheus.MustNewConstMetric(c.rpcCallCheckDesc, prometheus.CounterValue, float64(s.BadFmt), "bad_fmt")
	ch <- prometheus.MustNewConstMetric(c.rpcCallCheckDesc, prometheus.CounterValue, float64(s.BadAuth), "bad_auth")
	ch <- prometheus.MustNewConstMetric(c.rpcCallCheckDesc, prometheus.CounterValue, float64(s.BadClnt), "bad_clnt")
}

// updateNFSdRequestsv2Stats collects statistics for NFSv2 requests.
func (c *nfsdCollector) updateNFSdRequestsV2Stats(ch chan<- prometheus.Metric, s *nfs.V2stats) {
	if c.skipV2 {
		return
	}
	v := reflect.ValueOf(s).Elem()
	for i := int(s.Fields); i > 0; i-- {
		field := v.Field(i)
		ch <- prometheus.MustNewConstMetric(c.nfsV2callDesc, prometheus.CounterValue, float64(field.Uint()), v.Type().Field(i).Name)
	}
}

// updateNFSdRequestsv3Stats collects statistics for NFSv3 requests.
func (c *nfsdCollector) updateNFSdRequestsV3Stats(ch chan<- prometheus.Metric, s *nfs.V3stats) {
	if c.skipV3 {
		return
	}
	v := reflect.ValueOf(s).Elem()
	for i := int(s.Fields); i > 0; i-- {
		field := v.Field(i)
		ch <- prometheus.MustNewConstMetric(c.nfsV3callDesc, prometheus.CounterValue, float64(field.Uint()), v.Type().Field(i).Name)
	}
}

// updateNFSdRequestsv4Stats collects statistics for NFSv4 requests.
func (c *nfsdCollector) updateNFSdRequestsV4Stats(ch chan<- prometheus.Metric, s *nfs.V4statsServer) {
	if c.skipV4 {
		return
	}
	v := reflect.ValueOf(s).Elem()
	for i := int(s.Fields); i > 0; i-- {
		field := v.Field(i)
		ch <- prometheus.MustNewConstMetric(c.nfsV4callDesc, prometheus.CounterValue, float64(field.Uint()), v.Type().Field(i).Name)
	}
}

// updateNFSdRequestsV4Ops collects statistics for NFSv4 operations.
func (c *nfsdCollector) updateNFSdRequestsV4Ops(ch chan<- prometheus.Metric, s *nfs.V4ops) {
	if c.skipV4ops {
		return
	}
	v := reflect.ValueOf(s).Elem()
	for i := int(s.Fields); i > 2; i-- {
		field := v.Field(i)
		ch <- prometheus.MustNewConstMetric(c.nfsV4opDesc, prometheus.CounterValue, float64(field.Uint()), v.Type().Field(i).Name)
	}
}
