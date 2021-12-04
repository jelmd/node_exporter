// Copyright 2015 The Prometheus Authors
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

//go:build !nocpu
// +build !nocpu

package collector

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs"
	"github.com/prometheus/procfs/sysfs"
	"gopkg.in/alecthomas/kingpin.v2"
)

type cpuCollector struct {
	fs                 procfs.FS
	cpu                *prometheus.Desc
	cpuInfo            *prometheus.Desc
	cpuFlagsInfo       *prometheus.Desc
	cpuBugsInfo        *prometheus.Desc
	cpuGuest           *prometheus.Desc
	cpuCoreThrottle    *prometheus.Desc
	cpuPackageThrottle *prometheus.Desc
	logger             log.Logger
	cpuInfoLabels      []string
	cpuInfoValues      []string
	cpuFlagsInfoValues []string
	cpuBugsInfoValues  []string
	cpuStats           []procfs.CPUStat
	cpuStatsMutex      sync.Mutex

	cpuFlagsIncludeRegexp *regexp.Regexp
	cpuBugsIncludeRegexp  *regexp.Regexp
}

// Idle jump back limit in seconds.
const jumpBackSeconds = 3.0

var (
	enableCPUGuest       = kingpin.Flag("collector.cpu.guest", "Enables metric node_cpu_guest_seconds_total").Default("true").Bool()
	enableCPUInfo        = kingpin.Flag("collector.cpu.info", "Enables metric cpu_info").Bool()
	enableStats          = kingpin.Flag("collector.cpu.stats", "Enables metric cpu_seconds").Default("true").Bool()
	enableThermThrottle  = kingpin.Flag("collector.cpu.throttle", "Enables metric cpu_seconds").Default("true").Bool()
	flagsInclude         = kingpin.Flag("collector.cpu.info.flags-include", "Filter the `flags` field in cpuInfo with a value that must be a regular expression").String()
	bugsInclude          = kingpin.Flag("collector.cpu.info.bugs-include", "Filter the `bugs` field in cpuInfo with a value that must be a regular expression").String()
	jumpBackDebugMessage = fmt.Sprintf("CPU Idle counter jumped backwards more than %f seconds, possible hotplug event, resetting CPU stats", jumpBackSeconds)
)

func init() {
	registerCollector("cpu", defaultEnabled, NewCPUCollector)
}

// NewCPUCollector returns a new Collector exposing kernel/system statistics.
func NewCPUCollector(logger log.Logger) (Collector, error) {
	fs, err := procfs.NewFS(*procPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open procfs: %w", err)
	}
	info, err := fs.CPUInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get /proc/cpuinfo: %w", err)
	}

	// pre-initialize collector vars
	var cpuInfo, cpuFlagsInfo, cpuBugsInfo, cpuGuest, cpuCoreThrottle, cpuPackageThrottle *prometheus.Desc
	flagValues := make([]string, 0)
	bugValues := make([]string, 0)
	infoLabels := []string{ "package", "vendor", "family", "model", "model_name", "microcode", "stepping", "cachesize", "cores", "freq_base", "freq_max", "freq_min" }
	infoValues := make([]string, 0)

	if len(info) != 0 {
		cpu := info[0]
		if *flagsInclude != "" {
			level.Info(logger).Log("msg", "flagsInclude", "cpu", *flagsInclude)
			regex, err := regexp.Compile(*flagsInclude)
			if err != nil {
				return nil, fmt.Errorf("failed to compile --collector.cpu.info.flags-include, the values of them must be regular expressions: %w", err)
			}
			for _, val := range cpu.Flags {
				if regex.MatchString(val) {
					flagValues = append(flagValues, val)
				}
			}
			cpuFlagsInfo = prometheus.NewDesc(
				prometheus.BuildFQName(namespace, cpuCollectorSubsystem, "flag_info"),
				"The `flags` field of CPU information from /proc/cpuinfo taken from the first core. On change the collector needs to be restarted.",
				[]string{"flag"}, nil,
			)
		}

		if *bugsInclude != "" {
			level.Info(logger).Log("msg", "bugsInclude", "cpu", *bugsInclude)
			regex, err := regexp.Compile(*bugsInclude)
			if err != nil {
				return nil, fmt.Errorf("failed to compile --collector.cpu.info.bugs-include, the values of them must be regular expressions: %w", err)
			}
			for _, val := range cpu.Bugs {
				if regex.MatchString(val) {
					bugValues = append(bugValues, val)
				}
			}
			cpuBugsInfo = prometheus.NewDesc(
				prometheus.BuildFQName(namespace, cpuCollectorSubsystem, "bug_info"),
				"The `bugs` field of CPU information from /proc/cpuinfo taken from the first core. On change the collector needs to be restarted.",
				[]string{"bug"}, nil,
			)
		}
	}

	if *enableCPUInfo {
		sfs, err := sysfs.NewFS(*sysPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open sysfs: %w", err)
		}
		cpuFreqs, err := sfs.SystemCpufreq()
		if err != nil {
			return nil, fmt.Errorf("failed to get /sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_*_freq: %w", err)
		}
		var seen uint64 = 0
		var min, max, base, model string
		for _, cpu := range info {
			if (seen & (1 << cpu.PhysicalID)) != 0 {
				continue
			}
			seen |= (1 << cpu.PhysicalID)
			min = "0"
			max = "0"
			base = "0"
			// find the right cpu core
			if cpuFreqs != nil {
				for _, stats := range cpuFreqs {
					if stats.Name == cpu.CoreID {
						// TBD: scheinen vertauscht zu sein
						min = strconv.FormatUint(*stats.CpuinfoMinimumFrequency,10) + "000"
						max = strconv.FormatUint(*stats.CpuinfoMaximumFrequency,10) + "000"
						base, err = sfs.SystemCpuBaseFrequency(stats.Name)
						break
					}
				}
			}
			model = strings.Replace(cpu.ModelName, "(R)", "", -1)
			model = strings.Replace(model, " CPU", "", -1)
			model = strings.Replace(model, " Processor", "", -1)
			if strings.HasSuffix(model, "Hz") {
				idx := strings.LastIndexByte(model, ' ')
				if (idx > 0) {
					model = model[:idx-2]	// remove ' @' as well
				}
			}
			if strings.HasSuffix(model, "-Core") {
				idx := strings.LastIndexByte(model, ' ')
				if (idx > 0) {
					model = model[:idx]
				}
			}
			// same order as infoLabels
			infoValues = append(infoValues, strconv.Itoa(int(cpu.PhysicalID)))
			infoValues = append(infoValues, cpu.VendorID)
			infoValues = append(infoValues, cpu.CPUFamily)
			infoValues = append(infoValues, cpu.Model)
			infoValues = append(infoValues, model)
			infoValues = append(infoValues, cpu.Microcode)
			infoValues = append(infoValues, cpu.Stepping)
			infoValues = append(infoValues, cpu.CacheSize)
			infoValues = append(infoValues, strconv.Itoa(int(cpu.CPUCores)))
			infoValues = append(infoValues, base)
			infoValues = append(infoValues, max)
			infoValues = append(infoValues, min)
		}
		cpuInfo = prometheus.NewDesc(
			prometheus.BuildFQName(namespace, cpuCollectorSubsystem, "info"),
			"Cached /proc/cpuinfo and system/cpu/*/cpufreq/cpuinfo_{min,max}_freq per package. On change the collector needs to be restarted.",
			infoLabels, nil,
		)
	}
	if *enableCPUGuest {
		cpuGuest = prometheus.NewDesc(
			prometheus.BuildFQName(namespace, cpuCollectorSubsystem, "guest_seconds_total"),
			"Seconds the CPUs spent in guests (VMs) for each mode.",
			[]string{"cpu", "mode"}, nil,
		)
	}
	if *enableThermThrottle {
		cpuCoreThrottle = prometheus.NewDesc(
			prometheus.BuildFQName(namespace, cpuCollectorSubsystem, "core_throttles_total"),
			"Number of times this CPU core has been throttled.",
			[]string{"package", "core"}, nil,
		)
		cpuPackageThrottle = prometheus.NewDesc(
			prometheus.BuildFQName(namespace, cpuCollectorSubsystem, "package_throttles_total"),
			"Number of times this CPU package has been throttled.",
			[]string{"package"}, nil,
		)
	}

	c := &cpuCollector{
		fs:  fs,
		cpu: nodeCPUSecondsDesc,
		cpuInfoLabels: infoLabels,
		cpuInfoValues: infoValues,
		cpuFlagsInfoValues: flagValues,
		cpuBugsInfoValues: bugValues,
		cpuInfo: cpuInfo,
		cpuFlagsInfo: cpuFlagsInfo,
		cpuBugsInfo: cpuBugsInfo,
		cpuGuest: cpuGuest,
		cpuCoreThrottle: cpuCoreThrottle,
		cpuPackageThrottle: cpuPackageThrottle,
		logger: logger,
	}

	return c, nil
}

// Update implements Collector and exposes cpu related metrics from /proc/stat and /sys/.../cpu/.
func (c *cpuCollector) Update(ch chan<- prometheus.Metric) error {
	if err := c.updateInfo(ch); err != nil {
		return err
	}
	if *enableStats {
		if err := c.updateStat(ch); err != nil {
			return err
		}
	}
	if *enableThermThrottle {
		return c.updateThermalThrottle(ch)
	}
	return nil
}

func (c *cpuCollector) updateInfo(ch chan<- prometheus.Metric) error {
	last := len(c.cpuInfoValues)
	if last != 0 {
		k := len(c.cpuInfoLabels)
		for i := 0; i < last; i += k {
			ch <- prometheus.MustNewConstMetric(c.cpuInfo, prometheus.GaugeValue, 1, c.cpuInfoValues[i:i+k]...)
		}
	}

	if len(c.cpuFlagsInfoValues) != 0 {
		for _, val := range c.cpuFlagsInfoValues {
			ch <- prometheus.MustNewConstMetric(c.cpuFlagsInfo, prometheus.GaugeValue, 1, val,)
		}
	}
	if len(c.cpuBugsInfoValues) != 0 {
		for _, val := range c.cpuBugsInfoValues {
			ch <- prometheus.MustNewConstMetric(c.cpuBugsInfo, prometheus.GaugeValue, 1, val,)
		}
	}

	return nil
}

// updateThermalThrottle reads /sys/devices/system/cpu/cpu* and expose thermal throttle statistics.
func (c *cpuCollector) updateThermalThrottle(ch chan<- prometheus.Metric) error {
	cpus, err := filepath.Glob(sysFilePath("devices/system/cpu/cpu[0-9]*"))
	if err != nil {
		return err
	}

	packageThrottles := make(map[uint64]uint64)
	packageCoreThrottles := make(map[uint64]map[uint64]uint64)

	// cpu loop
	for _, cpu := range cpus {
		// See
		// https://www.kernel.org/doc/Documentation/x86/topology.txt
		// https://www.kernel.org/doc/Documentation/cputopology.txt
		// https://www.kernel.org/doc/Documentation/ABI/testing/sysfs-devices-system-cpu
		var err error
		var physicalPackageID, coreID uint64

		// topology/physical_package_id
		if physicalPackageID, err = readUintFromFile(filepath.Join(cpu, "topology", "physical_package_id")); err != nil {
			level.Debug(c.logger).Log("msg", "CPU is missing physical_package_id", "cpu", cpu)
			continue
		}
		// topology/core_id
		if coreID, err = readUintFromFile(filepath.Join(cpu, "topology", "core_id")); err != nil {
			level.Debug(c.logger).Log("msg", "CPU is missing core_id", "cpu", cpu)
			continue
		}

		// metric node_cpu_core_throttles_total
		//
		// We process this metric before the package throttles as there
		// are CPU+kernel combinations that only present core throttles
		// but no package throttles.
		// Seen e.g. on an Intel Xeon E5472 system with RHEL 6.9 kernel.
		if _, present := packageCoreThrottles[physicalPackageID]; !present {
			packageCoreThrottles[physicalPackageID] = make(map[uint64]uint64)
		}
		if _, present := packageCoreThrottles[physicalPackageID][coreID]; !present {
			// Read thermal_throttle/core_throttle_count only once
			if coreThrottleCount, err := readUintFromFile(filepath.Join(cpu, "thermal_throttle", "core_throttle_count")); err == nil {
				packageCoreThrottles[physicalPackageID][coreID] = coreThrottleCount
			} else {
				level.Debug(c.logger).Log("msg", "CPU is missing core_throttle_count", "cpu", cpu)
			}
		}

		// metric node_cpu_package_throttles_total
		if _, present := packageThrottles[physicalPackageID]; !present {
			// Read thermal_throttle/package_throttle_count only once
			if packageThrottleCount, err := readUintFromFile(filepath.Join(cpu, "thermal_throttle", "package_throttle_count")); err == nil {
				packageThrottles[physicalPackageID] = packageThrottleCount
			} else {
				level.Debug(c.logger).Log("msg", "CPU is missing package_throttle_count", "cpu", cpu)
			}
		}
	}

	for physicalPackageID, packageThrottleCount := range packageThrottles {
		ch <- prometheus.MustNewConstMetric(c.cpuPackageThrottle,
			prometheus.CounterValue,
			float64(packageThrottleCount),
			strconv.FormatUint(physicalPackageID, 10))
	}

	for physicalPackageID, coreMap := range packageCoreThrottles {
		for coreID, coreThrottleCount := range coreMap {
			ch <- prometheus.MustNewConstMetric(c.cpuCoreThrottle,
				prometheus.CounterValue,
				float64(coreThrottleCount),
				strconv.FormatUint(physicalPackageID, 10),
				strconv.FormatUint(coreID, 10))
		}
	}
	return nil
}

// updateStat reads /proc/stat through procfs and exports CPU-related metrics.
func (c *cpuCollector) updateStat(ch chan<- prometheus.Metric) error {
	stats, err := c.fs.Stat()
	if err != nil {
		return err
	}

	c.updateCPUStats(stats.CPU)

	// Acquire a lock to read the stats.
	c.cpuStatsMutex.Lock()
	defer c.cpuStatsMutex.Unlock()
	for cpuID, cpuStat := range c.cpuStats {
		cpuNum := strconv.Itoa(cpuID)
		ch <- prometheus.MustNewConstMetric(c.cpu, prometheus.CounterValue, cpuStat.User, cpuNum, "user")
		ch <- prometheus.MustNewConstMetric(c.cpu, prometheus.CounterValue, cpuStat.Nice, cpuNum, "nice")
		ch <- prometheus.MustNewConstMetric(c.cpu, prometheus.CounterValue, cpuStat.System, cpuNum, "system")
		ch <- prometheus.MustNewConstMetric(c.cpu, prometheus.CounterValue, cpuStat.Idle, cpuNum, "idle")
		ch <- prometheus.MustNewConstMetric(c.cpu, prometheus.CounterValue, cpuStat.Iowait, cpuNum, "iowait")
		ch <- prometheus.MustNewConstMetric(c.cpu, prometheus.CounterValue, cpuStat.IRQ, cpuNum, "irq")
		ch <- prometheus.MustNewConstMetric(c.cpu, prometheus.CounterValue, cpuStat.SoftIRQ, cpuNum, "softirq")
		ch <- prometheus.MustNewConstMetric(c.cpu, prometheus.CounterValue, cpuStat.Steal, cpuNum, "steal")

		if *enableCPUGuest {
			// Guest CPU is also accounted for in cpuStat.User and cpuStat.Nice, expose these as separate metrics.
			ch <- prometheus.MustNewConstMetric(c.cpuGuest, prometheus.CounterValue, cpuStat.Guest, cpuNum, "user")
			ch <- prometheus.MustNewConstMetric(c.cpuGuest, prometheus.CounterValue, cpuStat.GuestNice, cpuNum, "nice")
		}
	}

	return nil
}

// updateCPUStats updates the internal cache of CPU stats.
func (c *cpuCollector) updateCPUStats(newStats []procfs.CPUStat) {

	// Acquire a lock to update the stats.
	c.cpuStatsMutex.Lock()
	defer c.cpuStatsMutex.Unlock()

	// Reset the cache if the list of CPUs has changed.
	if len(c.cpuStats) != len(newStats) {
		c.cpuStats = make([]procfs.CPUStat, len(newStats))
	}

	for i, n := range newStats {
		// If idle jumps backwards by more than X seconds, assume we had a hotplug event and reset the stats for this CPU.
		if (c.cpuStats[i].Idle - n.Idle) >= jumpBackSeconds {
			level.Debug(c.logger).Log("msg", jumpBackDebugMessage, "cpu", i, "old_value", c.cpuStats[i].Idle, "new_value", n.Idle)
			c.cpuStats[i] = procfs.CPUStat{}
		}

		if n.Idle >= c.cpuStats[i].Idle {
			c.cpuStats[i].Idle = n.Idle
		} else {
			level.Debug(c.logger).Log("msg", "CPU Idle counter jumped backwards", "cpu", i, "old_value", c.cpuStats[i].Idle, "new_value", n.Idle)
		}

		if n.User >= c.cpuStats[i].User {
			c.cpuStats[i].User = n.User
		} else {
			level.Debug(c.logger).Log("msg", "CPU User counter jumped backwards", "cpu", i, "old_value", c.cpuStats[i].User, "new_value", n.User)
		}

		if n.Nice >= c.cpuStats[i].Nice {
			c.cpuStats[i].Nice = n.Nice
		} else {
			level.Debug(c.logger).Log("msg", "CPU Nice counter jumped backwards", "cpu", i, "old_value", c.cpuStats[i].Nice, "new_value", n.Nice)
		}

		if n.System >= c.cpuStats[i].System {
			c.cpuStats[i].System = n.System
		} else {
			level.Debug(c.logger).Log("msg", "CPU System counter jumped backwards", "cpu", i, "old_value", c.cpuStats[i].System, "new_value", n.System)
		}

		if n.Iowait >= c.cpuStats[i].Iowait {
			c.cpuStats[i].Iowait = n.Iowait
		} else {
			level.Debug(c.logger).Log("msg", "CPU Iowait counter jumped backwards", "cpu", i, "old_value", c.cpuStats[i].Iowait, "new_value", n.Iowait)
		}

		if n.IRQ >= c.cpuStats[i].IRQ {
			c.cpuStats[i].IRQ = n.IRQ
		} else {
			level.Debug(c.logger).Log("msg", "CPU IRQ counter jumped backwards", "cpu", i, "old_value", c.cpuStats[i].IRQ, "new_value", n.IRQ)
		}

		if n.SoftIRQ >= c.cpuStats[i].SoftIRQ {
			c.cpuStats[i].SoftIRQ = n.SoftIRQ
		} else {
			level.Debug(c.logger).Log("msg", "CPU SoftIRQ counter jumped backwards", "cpu", i, "old_value", c.cpuStats[i].SoftIRQ, "new_value", n.SoftIRQ)
		}

		if n.Steal >= c.cpuStats[i].Steal {
			c.cpuStats[i].Steal = n.Steal
		} else {
			level.Debug(c.logger).Log("msg", "CPU Steal counter jumped backwards", "cpu", i, "old_value", c.cpuStats[i].Steal, "new_value", n.Steal)
		}

		if *enableCPUGuest {
			if n.Guest >= c.cpuStats[i].Guest {
				c.cpuStats[i].Guest = n.Guest
			} else {
				level.Debug(c.logger).Log("msg", "CPU Guest counter jumped backwards", "cpu", i, "old_value", c.cpuStats[i].Guest, "new_value", n.Guest)
			}

			if n.GuestNice >= c.cpuStats[i].GuestNice {
				c.cpuStats[i].GuestNice = n.GuestNice
			} else {
				level.Debug(c.logger).Log("msg", "CPU GuestNice counter jumped backwards", "cpu", i, "old_value", c.cpuStats[i].GuestNice, "new_value", n.GuestNice)
			}
		}
	}
}
