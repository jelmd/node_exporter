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

	"github.com/prometheus/procfs/internal/util"
)

// ParseClientRPCStats returns stats read from /proc/net/rpc/nfs
func ParseProcNetRpcNfsStats(r io.Reader) (*ProcNetRpcNfsStats, error) {
	stats := &ProcNetRpcNfsStats{}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(scanner.Text())
		// require at least <key> <value>
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid NFS metric line %q", line)
		}

		label := parts[0]
		if label == "net" {
			continue
		}
		min := 0
		if label == "proc4" {
			min = LAST_NFS4_CLNT_OP + 2
		}

		values, err := util.ParseUint64s(parts[1:], min)
		if err != nil {
			return nil, fmt.Errorf("error parsing NFS metric line: %w", err)
		}

		switch label {
		case "rpc":
			stats.RpcClient, err = parseRpcClient(values)
		case "proc2":
			stats.V2stats, err = parseV2stats(values)
		case "proc3":
			stats.V3stats, err = parseV3stats(values)
		case "proc4":
			stats.V4statsClient, err = parseV4statsClient(values)
		default:
			return nil, fmt.Errorf("unknown NFS metric line %q", label)
		}
		if err != nil {
			return nil, fmt.Errorf("errors parsing NFS metric line: %w", err)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning NFS file: %w", err)
	}

	return stats, nil
}
