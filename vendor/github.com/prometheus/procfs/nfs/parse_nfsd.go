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

package nfs

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"strconv"

	"github.com/prometheus/procfs/internal/util"
)

// ParseServerRPCStats returns stats read from /proc/net/rpc/nfsd
func ParseProcNetRpcNfsdStats(r io.Reader) (*ProcNetRpcNfsdStats, error) {
	stats := &ProcNetRpcNfsdStats{}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(scanner.Text())
		// require at least <key> <value>
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid NFSd metric line %q", line)
		}
		label := parts[0]

		var values []uint64
		var err error
		if label == "ra" {
			continue
		}
		min := 0
		if label == "th" || label == "fh" {
			if len(parts) < 3 {
				return nil, fmt.Errorf("invalid NFSd th metric line %q", line)
			}
			u, err := strconv.ParseUint(parts[1], 10, 64)
			if err == nil {
				if label == "th" {
					stats.Threads = Threads{ Threads: u, }
				} else {
					stats.FileHandles = FileHandles{ Stale: u, }
				}
				continue
			}
		} else {
			if label == "proc4ops" {
				min = LAST_NFS4_OP + 2
			}
			values, err = util.ParseUint64s(parts[1:], min)
		}
		if err != nil {
			return nil, fmt.Errorf("error parsing NFSd metric line %s: %w", label, err)
		}

		switch label {
		case "rc":
			stats.ReplyCache, err = parseReplyCache(values)
		case "io":
			stats.InputOutput, err = parseInputOutput(values)
		case "net":
			stats.Network, err = parseNetwork(values)
		case "rpc":
			stats.RpcServer, err = parseRpcServer(values)
		case "proc2":
			stats.V2stats, err = parseV2stats(values)
		case "proc3":
			stats.V3stats, err = parseV3stats(values)
		case "proc4":
			stats.V4statsServer, err = parseV4statsServer(values)
		case "proc4ops":
			stats.V4ops, err = parseV4ops(values)
		default:
			return nil, fmt.Errorf("unknown NFSd metric line %q", label)
		}
		if err != nil {
			return nil, fmt.Errorf("errors parsing NFSd metric line: %w", err)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning NFSd file: %w", err)
	}

	return stats, nil
}
