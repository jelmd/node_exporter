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

// Package nfs implements parsing of /proc/net/rpc/nfs[d].
// See also:
// - https://github.com/torvalds/linux/blob/master/include/linux/nfs4.h
// - https://github.com/torvalds/linux/blob/master/fs/nfsd/stats.h
// - https://github.com/torvalds/linux/blob/master/fs/nfsd/stats.c
// - https://github.com/torvalds/linux/blob/master/net/sunrpc/stats.c
package nfs

import (
	"os"
	"strings"

	"github.com/prometheus/procfs/internal/fs"
)

// ReplyCache models the "rc" line.
type ReplyCache struct {
	Hits    uint64
	Misses  uint64
	NoCache uint64
}

// FileHandles models the "fh" line. Deprecated.
type FileHandles struct {
	Stale        uint64
	TotalLookups uint64		// on Linux always 0
	AnonLookups  uint64		// on Linux always 0
	DirNoCache   uint64		// on Linux always 0
	NoDirNoCache uint64		// on Linux always 0
}

// InputOutput models the "io" line.
type InputOutput struct {
	Read  uint64
	Write uint64
}

// Threads models the "th" line. Deprecated.
type Threads struct {
	Threads uint64			// static
	FullCnt uint64			// on Linux always 0
}

// ReadAheadCache models the "ra" line. Deprecated.
type ReadAheadCache struct {
	CacheSize      uint64	// on Linux always 0
	CacheHistogram []uint64	// on Linux always 0
	NotFound       uint64	// on Linux always 0
}


// Network models the "net" line. Generic SUN RPC stats.
type Network struct {
	NetCount   uint64
	UDPCount   uint64
	TCPCount   uint64
	TCPConnect uint64
}

// RpcServer models the nfsd "rpc" line.
type RpcServer struct {
	Good     uint64
	Bad      uint64		// sum of BadFmt + BadAuth + BadClnt
	BadFmt   uint64
	BadAuth  uint64
	BadClnt  uint64		// unused
}

// RpcClient models the nfs "rpc" line.
type RpcClient struct {
	RPCCount        uint64
	Retransmissions uint64
	AuthRefreshes   uint64
}


// V2stats models the "proc2" line.
type V2stats struct {
	Fields   uint64
	Null     uint64
	GetAttr  uint64
	SetAttr  uint64
	Root     uint64
	Lookup   uint64
	ReadLink uint64
	Read     uint64
	WriteCache  uint64
	Write    uint64
	Create   uint64
	Remove   uint64
	Rename   uint64
	Link     uint64
	SymLink  uint64
	MkDir    uint64
	RmDir    uint64
	ReadDir  uint64
	StatFs   uint64		// == 17
}
const LAST_NFS2_OP int = 17

// V3stats models the "proc3" line.
type V3stats struct {
	Fields      uint64
	Null        uint64
	GetAttr     uint64
	SetAttr     uint64
	Lookup      uint64
	Access      uint64
	ReadLink    uint64
	Read        uint64
	Write       uint64
	Create      uint64
	MkDir       uint64
	SymLink     uint64
	MkNod       uint64
	Remove      uint64
	RmDir       uint64
	Rename      uint64
	Link        uint64
	ReadDir     uint64
	ReadDirPlus uint64
	FsStat      uint64
	FsInfo      uint64
	PathConf    uint64
	Commit      uint64	// == 21
}
const LAST_NFS3_OP int = 21

// V4statsClient models the nfs "proc4" line.
type V4statsClient struct {
	Fields             uint64
	Null               uint64
	Read               uint64
	Write              uint64
	Commit             uint64
	Open               uint64
	OpenConfirm        uint64
	OpenNoAttr         uint64
	OpenDowngrade      uint64
	Close              uint64
	SetAttr            uint64
	FsInfo             uint64
	Renew              uint64
	SetClientId        uint64
	SetClientIdConfirm uint64
	Lock               uint64
	LockT              uint64
	LockU              uint64
	Access             uint64
	GetAttr            uint64
	Lookup             uint64
	LookupRoot         uint64
	Remove             uint64
	Rename             uint64
	Link               uint64
	Symlink            uint64
	Create             uint64
	Pathconf           uint64
	StatFs             uint64
	ReadLink           uint64
	ReadDir            uint64
	ServerCaps         uint64
	DelegReturn        uint64
	GetACL             uint64
	SetACL             uint64
	FsLocations        uint64
	ReleaseLockOwner   uint64
	SecInfo            uint64
	FsIdPresent        uint64

	// 4.1
	ExchangeId         uint64
	CreateSession      uint64
	DestroySession     uint64
	Sequence           uint64
	GetLeaseTime       uint64
	ReclaimComplete    uint64
	LayoutGet          uint64	// pNFS
	GetDeviceInfo      uint64	// pNFS
	LayoutCommit       uint64	// pNFS
	LayoutReturn       uint64	// pNFS
	SecInfoNoName      uint64
	TestStateId        uint64
	FreeStateId        uint64
	GetDeviceList      uint64
	BindConnToSession  uint64
	DestroyClientId    uint64

	// 4.2
	Seek               uint64
	Allocate           uint64
	DeAllocate         uint64
	LayoutStats        uint64
	Clone              uint64	// == 58
	Copy               uint64
	OffloadCancel      uint64
	LookupP            uint64
	LayoutError        uint64
	CopyNotify         uint64	// == 63

	// xattr support (RFC8276) - usually not included in the proc4 stats
	GetXattr           uint64
	SetXattr           uint64
	ListXattrs         uint64
	RemoveXattr        uint64
	ReadPlus           uint64	// == 68
}
const LAST_NFS4_CLNT_OP int = 68

// V4statsServer models the nfsd "proc4" line.
type V4statsServer struct {
	Fields   uint64
	Null     uint64
	Compound uint64
}

// V4ops models the "proc4ops" line: NFSv4 operations.
// Depending on the NFS version in use not all fields get used.
type V4ops struct {
	Fields       uint64			// number of fields in this record
	Unused0      uint64			// unused
	Unused1      uint64			// unused
	Unused2      uint64			// unused
	Access       uint64			// == 3		==	FIRST_NFS4_OP
	Close        uint64
	Commit       uint64
	Create       uint64
	DelegPurge   uint64			// unused
	DelegReturn  uint64
	GetAttr      uint64
	GetFH        uint64
	Link         uint64
	Lock         uint64
	LockT        uint64
	LockU        uint64
	Lookup       uint64
	LookupP      uint64
	Nverify      uint64
	Open         uint64
	OpenAttr     uint64			// unused
	OpenConfirm  uint64
	OpenDowngrade      uint64
	PutFH        uint64
	PutPubFH     uint64
	PutRootFH    uint64
	Read         uint64
	ReadDir      uint64
	ReadLink     uint64
	Remove       uint64
	Rename       uint64
	Renew        uint64
	RestoreFH    uint64
	SaveFH       uint64
	SecInfo      uint64
	SetAttr      uint64
	SetClientId  uint64
	SetClientIdConfirm uint64
	Verify       uint64
	Write        uint64
	ReleaseLockOwner   uint64	// == 39	==	LAST_NFS40_OP

	// 4.1
	BackChannelCtl     uint64
	BindConnToSession  uint64
	ExchangeId         uint64
	CreateSession      uint64
	DestroySession     uint64
	FreeStateId        uint64
	GetDirDelegation   uint64	// unused
	GetDeviceInfo      uint64	// pNFS
	GetDeviceList      uint64
	LayoutCommit       uint64	// pNFS
	LayoutGet          uint64	// pNFS
	LayoutReturn       uint64	// pNFS
	SecInfoNoName      uint64
	Sequence           uint64
	SetSSV             uint64	// unused
	TestStateId        uint64
	WantDelegation     uint64
	DestroyClientId    uint64
	ReclaimComplete    uint64	// == 58	==	LAST_NFS41_OP

	// 4.2
	Allocate           uint64
	Copy               uint64
	CopyNotify         uint64
	DeAllocate         uint64
	IoAdvise           uint64	// unused
	LayoutError        uint64	// unused
	LayoutStats        uint64	// unused
	OffloadCancel      uint64
	OffloadStatus      uint64
	ReadPlus           uint64
	Seek               uint64
	WriteSame          uint64	// unused
	Clone              uint64	// == 71

	// xattr support (RFC8276)
	GetXattr           uint64
	SetXattr           uint64
	ListXattrs         uint64
	RemoveXattr        uint64	// == 75	==  LAST_NFS42_OP   == LAST_NFS4_OP
}
const LAST_NFS4_OP int = 75

// ClientStats from /proc/net/rpc/nfs.
type ProcNetRpcNfsStats struct {
	RpcClient       RpcClient
	V2stats         V2stats
	V3stats         V3stats
	V4statsClient   V4statsClient
}

// ServerStats from /proc/net/rpc/nfsd.
type ProcNetRpcNfsdStats struct {
	ReplyCache     ReplyCache
	FileHandles    FileHandles
	InputOutput    InputOutput
	Threads        Threads
	Network        Network
	RpcServer      RpcServer
	V2stats        V2stats
	V3stats        V3stats
	V4statsServer  V4statsServer
	V4ops          V4ops
}

// FS represents the pseudo-filesystem proc, which provides an interface to
// kernel data structures.
type FS struct {
	proc *fs.FS
}

// NewDefaultFS returns a new FS mounted under the default mountPoint. It will error
// if the mount point can't be read.
func NewDefaultFS() (FS, error) {
	return NewFS(fs.DefaultProcMountPoint)
}

// NewFS returns a new FS mounted under the given mountPoint. It will error
// if the mount point can't be read.
func NewFS(mountPoint string) (FS, error) {
	if strings.TrimSpace(mountPoint) == "" {
		mountPoint = fs.DefaultProcMountPoint
	}
	fs, err := fs.NewFS(mountPoint)
	if err != nil {
		return FS{}, err
	}
	return FS{&fs}, nil
}

// Get stats from proc/net/rpc/nfs.
func (fs FS) ProcNetRpcNfsStats() (*ProcNetRpcNfsStats, error) {
	f, err := os.Open(fs.proc.Path("net/rpc/nfs"))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return ParseProcNetRpcNfsStats(f)
}

// Get stats from proc/net/rpc/nfsd.
func (fs FS) ProcNetRpcNfsdStats() (*ProcNetRpcNfsdStats, error) {
	f, err := os.Open(fs.proc.Path("net/rpc/nfsd"))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return ParseProcNetRpcNfsdStats(f)
}
