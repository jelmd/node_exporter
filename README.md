# Node exporter

This is a fork of [Node exporter](https://github.com/prometheus/node_exporter),
an metrics collector written in Go, which exposes several kernel and hardware
data of \*NIX OS in [Prometheus exposition format](https://prometheus.io/docs/instrumenting/exposition_formats/).

You can get more detailed information about it in the [upstream README](https://github.com/prometheus/node_exporter/blob/master/README.md).

In addition to the upstream this version has the following

## Enhancements
- The binary got renamed to _node-exporter_, which is easier to type at least on german layout keyboards and allows one to install it side-by-side with the original.
- New _collector.cpus_ - it exposes the number of CPU cores (or strands if HT or SMT is enabled) currently on- and offline.
- _collector.nfs_, _collector.nfsd_ (Linux):
    - Cleanup, fix and consolidation.
    - Added support for NFS 4.1 and 4.2 incl. RFC 8276 operations.
    - NFS metrics got renamed to something, what makes sense to admins.
    - New feature _collector.nfsd.skip=list_ - allows to turn off parsinging and exposing nfsd metrics for the given list of NFS versions.
    - The _collector.nfsd_ now exposes /proc/fs/nfsd/pool\_stats metrics as well. If you have any NFS problems, these are the metrics you should check first.
- _collector.pressure_ (Linux):
    - Misleading/vague HELP messages got replaced, are now kernel documentation conform. 
    - Metrics got renamed to _psi_ (instead of pressure) and labels are now kernel documentation conform.
    - Values get exposed as is in µs, are not converted to seconds anymore.
- _collector.cpu_:
    - New options _--no-collector.cpu.stats_ and _--no-collector.cpu.throttle_ options can be used to disable (or w/o _no-_ to explicitly enable) collecting and exposing a lot of CPU related metrics, which are in a day-by-day monitoring more or less useless (especially if one has many cores CPUs). 
    - _collector.cpu.info_ optimization: /proc/cpuinfo gets parsed only once, when the collector gets initialized because it is unlikely to change. Furthermore  data are now collected per CPU package and not per hyperthread/strand. This reduces redundant data and the metrics cardinality especially for many core CPUs a lot.
    - _collcetor.cpu.info_: Useless bloat gets removed from model\_name and min, max and base frequency provided in a separate label entry. 
- _collector.dmi_: HELP message got replaced with a shorter description which makes in addition sense.
- New option _--compact_: disable sending TYPE and HELP messages for each metric. Reduces the transmitted volume up to 60% depending on what is enabled.
- New option _--web.disable-go-metrics_: Usually only Go developers (and often not even those) can possibly deduce something useful from go metrics. So admins now have an option to reduce this bloat to zero for day-by-day monitoring.
- New feature: *node\_scrape\_collector\_duration\_seconds{collector="overall"}* shows the time it took to obtain and format data from all collectors (can happen concurrently, so not necessarily the sum of all collector scrapetimes).
- The version string is now completely human readable - useless VCS infos dropped.
- Build:
  - The default target is now _build_.
  - Vendor files are now tracked as well. They get patched as needed so
    take care when running a 'go mod vendor'.

## Download
You may download a [debian package](https://pkg.cs.ovgu.de/LNF/linux/ubuntu/20.04/) made for Ubuntu 20.04 using this URL https://pkg.cs.ovgu.de/LNF/linux/ubuntu/20.04/. It is self-containing and should work out of the box. It should work on other Linux Distros as well, but has not been tested there. You may use alien or an archive manager of your choice to extract the binary and systemd related files as needed if your distro is not debian package driven.
