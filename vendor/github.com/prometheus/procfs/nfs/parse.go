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
	"fmt"
)

func parseReplyCache(v []uint64) (ReplyCache, error) {
	if len(v) != 3 {
		return ReplyCache{}, fmt.Errorf("invalid rc line %q", v)
	}

	return ReplyCache{
		Hits:    v[0],
		Misses:  v[1],
		NoCache: v[2],
	}, nil
}

func parseInputOutput(v []uint64) (InputOutput, error) {
	if len(v) != 2 {
		return InputOutput{}, fmt.Errorf("invalid io line %q", v)
	}

	return InputOutput{
		Read:  v[0],
		Write: v[1],
	}, nil
}

func parseNetwork(v []uint64) (Network, error) {
	if len(v) != 4 {
		return Network{}, fmt.Errorf("invalid net line %q", v)
	}

	return Network{
		NetCount:   v[0],
		UDPCount:   v[1],
		TCPCount:   v[2],
		TCPConnect: v[3],
	}, nil
}

func parseRpcServer(v []uint64) (RpcServer, error) {
	if len(v) != 5 {
		return RpcServer{}, fmt.Errorf("invalid rpc line %q", v)
	}

	return RpcServer{
		Good:     v[0],
		Bad:      v[1],
		BadFmt:   v[2],
		BadAuth:  v[3],
		BadClnt:  v[4],
	}, nil
}

func parseRpcClient(v []uint64) (RpcClient, error) {
	if len(v) != 3 {
		return RpcClient{}, fmt.Errorf("invalid RPC line %q", v)
	}

	return RpcClient{
		RPCCount:        v[0],
		Retransmissions: v[1],
		AuthRefreshes:   v[2],
	}, nil
}

func parseV2stats(v []uint64) (V2stats, error) {
	values := int(v[0])
	if len(v[1:]) != values || values < 18 {
		return V2stats{}, fmt.Errorf("invalid proc2 line %q", v)
	}

	return V2stats{
		Fields:   v[0],
		Null:     v[1],
		GetAttr:  v[2],
		SetAttr:  v[3],
		Root:     v[4],
		Lookup:   v[5],
		ReadLink: v[6],
		Read:     v[7],
		WriteCache:  v[8],
		Write:    v[9],
		Create:   v[10],
		Remove:   v[11],
		Rename:   v[12],
		Link:     v[13],
		SymLink:  v[14],
		MkDir:    v[15],
		RmDir:    v[16],
		ReadDir:  v[17],
		StatFs:   v[18],
	}, nil
}

func parseV3stats(v []uint64) (V3stats, error) {
	values := int(v[0])
	if len(v[1:]) != values || values < 22 {
		return V3stats{}, fmt.Errorf("invalid proc3 line %q", v)
	}

	return V3stats{
		Fields:      v[0],
		Null:        v[1],
		GetAttr:     v[2],
		SetAttr:     v[3],
		Lookup:      v[4],
		Access:      v[5],
		ReadLink:    v[6],
		Read:        v[7],
		Write:       v[8],
		Create:      v[9],
		MkDir:       v[10],
		SymLink:     v[11],
		MkNod:       v[12],
		Remove:      v[13],
		RmDir:       v[14],
		Rename:      v[15],
		Link:        v[16],
		ReadDir:     v[17],
		ReadDirPlus: v[18],
		FsStat:      v[19],
		FsInfo:      v[20],
		PathConf:    v[21],
		Commit:      v[22],
	}, nil
}

func parseV4statsClient(v []uint64) (V4statsClient, error) {
	values := int(v[0])
	if len(v) <= 69 || values < 36 {
		return V4statsClient{}, fmt.Errorf("invalid proc4 line (vals: %d, capacity: %d): %#v", values, len(v), v)
	}

	return V4statsClient{
		Fields:				v[0],
		Null:               v[1],
		Read:               v[2],
		Write:              v[3],
		Commit:             v[4],
		Open:               v[5],
		OpenConfirm:        v[6],
		OpenNoAttr:         v[7],
		OpenDowngrade:      v[8],
		Close:              v[9],
		SetAttr:            v[10],
		FsInfo:             v[11],
		Renew:              v[12],
		SetClientId:        v[13],
		SetClientIdConfirm: v[14],
		Lock:               v[15],
		LockT:              v[16],
		LockU:              v[17],
		Access:             v[18],
		GetAttr:            v[19],
		Lookup:             v[20],
		LookupRoot:         v[21],
		Remove:             v[22],
		Rename:             v[23],
		Link:               v[24],
		Symlink:            v[25],
		Create:             v[26],
		Pathconf:           v[27],
		StatFs:             v[28],
		ReadLink:           v[29],
		ReadDir:            v[30],
		ServerCaps:         v[31],
		DelegReturn:        v[32],
		GetACL:             v[33],
		SetACL:             v[34],
		FsLocations:        v[35],
		ReleaseLockOwner:   v[36],
		SecInfo:            v[37],
		FsIdPresent:        v[38],
		ExchangeId:         v[39],
		CreateSession:      v[40],
		DestroySession:     v[41],
		Sequence:           v[42],
		GetLeaseTime:       v[43],
		ReclaimComplete:    v[44],
		LayoutGet:          v[45],
		GetDeviceInfo:      v[46],
		LayoutCommit:       v[47],
		LayoutReturn:       v[48],
		SecInfoNoName:      v[49],
		TestStateId:        v[50],
		FreeStateId:        v[51],
		GetDeviceList:      v[52],
		BindConnToSession:  v[53],
		DestroyClientId:    v[54],
		Seek:               v[55],
		Allocate:           v[56],
		DeAllocate:         v[57],
		LayoutStats:        v[58],
		Clone:              v[59],
		Copy:               v[60],
		OffloadCancel:      v[61],
		LookupP:            v[62],
		LayoutError:        v[63],
		CopyNotify:         v[64],
		GetXattr:           v[65],
		SetXattr:           v[66],
		ListXattrs:         v[67],
		RemoveXattr:        v[68],
		ReadPlus:           v[69],
	}, nil
}

func parseV4statsServer(v []uint64) (V4statsServer, error) {
	values := int(v[0])
	if len(v[1:]) != values || values != 2 {
		return V4statsServer{}, fmt.Errorf("invalid proc4 line %q", v)
	}

	return V4statsServer{
		Fields:   v[0],
		Null:     v[1],
		Compound: v[2],
	}, nil
}

func parseV4ops(v []uint64) (V4ops, error) {
	values := int(v[0])
	if len(v) <= 76 || values < 40 {
		return V4ops{}, fmt.Errorf("invalid proc4ops line (vals: %d, capacity: %d): %#v", values, len(v), v)
	}

	stats := V4ops{
		Fields:       v[0],
		Unused0:      v[1],
		Unused1:      v[2],
		Unused2:      v[3],
		Access:       v[4],
		Close:        v[5],
		Commit:       v[6],
		Create:       v[7],
		DelegPurge:   v[8],
		DelegReturn:  v[9],
		GetAttr:      v[10],
		GetFH:        v[11],
		Link:         v[12],
		Lock:         v[13],
		LockT:        v[14],
		LockU:        v[15],
		Lookup:       v[16],
		LookupP:      v[17],
		Nverify:      v[18],
		Open:         v[19],
		OpenAttr:     v[20],
		OpenConfirm:  v[21],
		OpenDowngrade: v[22],
		PutFH:        v[23],
		PutPubFH:     v[24],
		PutRootFH:    v[25],
		Read:         v[26],
		ReadDir:      v[27],
		ReadLink:     v[28],
		Remove:       v[29],
		Rename:       v[30],
		Renew:        v[31],
		RestoreFH:    v[32],
		SaveFH:       v[33],
		SecInfo:      v[34],
		SetAttr:      v[35],
		SetClientId:        v[36],
		SetClientIdConfirm: v[37],
		Verify:             v[38],
		Write:              v[39],
		ReleaseLockOwner:   v[40],
		BackChannelCtl:     v[41],
		BindConnToSession:  v[42],
		ExchangeId:         v[43],
		CreateSession:      v[44],
		DestroySession:     v[45],
		FreeStateId:        v[46],
		GetDirDelegation:	v[47],
		GetDeviceInfo:      v[48],
		GetDeviceList:      v[49],
		LayoutCommit:       v[50],
		LayoutGet:          v[51],
		LayoutReturn:       v[52],
		SecInfoNoName:      v[53],
		Sequence:           v[54],
		SetSSV:             v[55],
		TestStateId:        v[56],
		WantDelegation:     v[57],
		DestroyClientId:    v[58],
		ReclaimComplete:    v[59],
		Allocate:           v[60],
		Copy:               v[61],
		CopyNotify:         v[62],
		DeAllocate:			v[63],
		IoAdvise:           v[64],
		LayoutError:        v[65],
		LayoutStats:        v[66],
		OffloadCancel:      v[67],
		OffloadStatus:      v[68],
		ReadPlus:           v[69],
		Seek:               v[70],
		WriteSame:          v[71],
		Clone:				v[72],
		GetXattr:			v[73],
		SetXattr:			v[74],
		ListXattrs:			v[75],
		RemoveXattr:		v[76],
	}

	return stats, nil
}
