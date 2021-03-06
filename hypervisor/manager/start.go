package manager

import (
	"bytes"
	"crypto/rand"
	"errors"
	"os"
	"path"
	"runtime"
	"syscall"

	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/log/prefixlogger"
	"github.com/Symantec/Dominator/lib/meminfo"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

const (
	dirPerms = syscall.S_IRWXU | syscall.S_IRGRP | syscall.S_IXGRP |
		syscall.S_IROTH | syscall.S_IXOTH
	privateFilePerms = syscall.S_IRUSR | syscall.S_IWUSR
	publicFilePerms  = privateFilePerms | syscall.S_IRGRP | syscall.S_IROTH
)

func newManager(startOptions StartOptions) (*Manager, error) {
	memInfo, err := meminfo.GetMemInfo()
	if err != nil {
		return nil, err
	}
	importCookie := make([]byte, 32)
	if _, err := rand.Read(importCookie); err != nil {
		return nil, err
	}
	err = fsutil.CopyToFile(path.Join(startOptions.StateDir, "import-cookie"),
		privateFilePerms, bytes.NewReader(importCookie), 0)
	if err != nil {
		return nil, err
	}
	manager := &Manager{
		StartOptions:      startOptions,
		importCookie:      importCookie,
		memTotalInMiB:     memInfo.Total >> 20,
		notifiers:         make(map[<-chan proto.Update]chan<- proto.Update),
		numCPU:            runtime.NumCPU(),
		vms:               make(map[string]*vmInfoType),
		volumeDirectories: startOptions.VolumeDirectories,
	}
	if err := manager.loadSubnets(); err != nil {
		return nil, err
	}
	if err := manager.loadAddressPool(); err != nil {
		return nil, err
	}
	dirname := path.Join(manager.StateDir, "VMs")
	dir, err := os.Open(dirname)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.Mkdir(dirname, dirPerms); err != nil {
				return nil, errors.New(
					"error making: " + dirname + ": " + err.Error())
			}
			dir, err = os.Open(dirname)
		}
	}
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	names, err := dir.Readdirnames(-1)
	if err != nil {
		return nil, errors.New(
			"error reading directory: " + dirname + ": " + err.Error())
	}
	for _, ipAddr := range names {
		vmDirname := path.Join(dirname, ipAddr)
		filename := path.Join(vmDirname, "info.json")
		var vmInfo vmInfoType
		if err := json.ReadFromFile(filename, &vmInfo); err != nil {
			return nil, err
		}
		vmInfo.Address.Shrink()
		vmInfo.manager = manager
		vmInfo.dirname = vmDirname
		vmInfo.ipAddress = ipAddr
		vmInfo.ownerUsers = make(map[string]struct{}, len(vmInfo.OwnerUsers))
		for _, username := range vmInfo.OwnerUsers {
			vmInfo.ownerUsers[username] = struct{}{}
		}
		vmInfo.logger = prefixlogger.New(ipAddr+": ", manager.Logger)
		vmInfo.metadataChannels = make(map[chan<- string]struct{})
		manager.vms[ipAddr] = &vmInfo
		if _, err := vmInfo.startManaging(0, false); err != nil {
			manager.Logger.Println(err)
			if ipAddr == "0.0.0.0" {
				delete(manager.vms, ipAddr)
				vmInfo.destroy()
			}
		}
	}
	// Check address pool for used addresses with no VM.
	freeIPs := make(map[string]struct{}, len(manager.addressPool.Free))
	for _, addr := range manager.addressPool.Free {
		freeIPs[addr.IpAddress.String()] = struct{}{}
	}
	for _, addr := range manager.addressPool.Registered {
		ipAddr := addr.IpAddress.String()
		if _, ok := freeIPs[ipAddr]; ok {
			continue
		}
		if _, ok := manager.vms[ipAddr]; !ok {
			manager.Logger.Printf("%s shown as used but no corresponding VM\n",
				ipAddr)
		}
	}
	if len(manager.volumeDirectories) < 1 {
		manager.volumeDirectories, err = getVolumeDirectories()
		if err != nil {
			return nil, err
		}
	}
	if len(manager.volumeDirectories) < 1 {
		return nil, errors.New("no volume directories available")
	}
	for _, volumeDirectory := range manager.volumeDirectories {
		if err := os.MkdirAll(volumeDirectory, dirPerms); err != nil {
			return nil, err
		}
	}
	return manager, nil
}
