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

package procfs

// The PSI / pressure interface is described at
//   https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git/tree/Documentation/accounting/psi.txt
// Each resource (cpu, io, memory, ...) is exposed as a single file.
// Each file may contain up to two lines, one for "some" pressure and one for "full" pressure.
// Each line contains several averages (over n seconds) and a total in µs.
//
// Example io pressure file:
// > some avg10=0.06 avg60=0.21 avg300=0.99 total=8537362
// > full avg10=0.00 avg60=0.13 avg300=0.96 total=8183134

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/prometheus/procfs/internal/util"
)

// PSIStats represent pressure stall information from /proc/pressure/*
// Some indicates the share of time in which at least some tasks are stalled
// Full indicates the share of time in which all non-idle tasks are stalled simultaneously
type PSIStats struct {
	Some int64
	Full int64
}

// PSIStatsForResource reads pressure stall information for the specified
// resource from /proc/pressure/<resource>. At time of writing this can be
// either "cpu", "memory" or "io".
func (fs FS) PSIStatsForResource(resource string) (PSIStats, error) {
	data, err := util.ReadFileNoStat(fs.proc.Path(fmt.Sprintf("%s/%s", "pressure", resource)))
	if err != nil {
		return PSIStats{}, fmt.Errorf("psi_stats: unavailable for %q: %w", resource, err)
	}

	return parsePSIStats(resource, bytes.NewReader(data))
}

// parsePSIStats parses the specified file for pressure stall information
func parsePSIStats(resource string, r io.Reader) (PSIStats, error) {
	psiStats := PSIStats{}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		s := scanner.Text()
		i := strings.LastIndexByte(s, '=')
		if i == -1 {
			continue
		}
		val, err := strconv.ParseInt(s[i+1:], 10, 64)
		if err != nil {
			return psiStats, err
		}
		if strings.HasPrefix(s, "some ") {
			psiStats.Some = val
		} else if strings.HasPrefix(s, "full ") {
			psiStats.Full = val
		}
		// If we encounter a line with an unknown prefix, ignore it and move on
	}

	return psiStats, nil
}
