package fsstorer

import (
	"bufio"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/Symantec/Dominator/lib/fsutil"
)

var zeroIP = IP{}

func (s *Storer) load() error {
	if err := s.readDirectory(nil, s.topDir); err != nil {
		return err
	}
	for hypervisor, ipAddrs := range s.hypervisorToIPs {
		for _, ipAddr := range ipAddrs {
			s.ipToHypervisor[ipAddr] = hypervisor
		}
	}
	return nil
}

func (s *Storer) readDirectory(partialIP []byte, dirname string) error {
	if len(partialIP) == len(zeroIP) {
		filename := filepath.Join(dirname, "ip-list.raw")
		if file, err := os.Open(filename); err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		} else {
			defer file.Close()
			reader := bufio.NewReader(file)
			var hyperAddr IP
			copy(hyperAddr[:], partialIP)
			var ipList []IP
			for {
				var ip IP
				if nRead, err := reader.Read(ip[:]); err != nil {
					if err == io.EOF {
						break
					}
					return err
				} else if nRead != len(ip) {
					return errors.New("incomplete read of IP address")
				}
				ipList = append(ipList, ip)
			}
			s.hypervisorToIPs[hyperAddr] = ipList
		}
		return nil
	}
	names, err := fsutil.ReadDirnames(dirname, true)
	if err != nil {
		return err
	}
	for _, name := range names {
		if val, err := strconv.ParseUint(name, 10, 8); err != nil {
			continue
		} else {
			moreIP := make([]byte, len(partialIP), len(partialIP)+1)
			copy(moreIP, partialIP)
			moreIP = append(moreIP, byte(val))
			subdir := filepath.Join(dirname, name)
			if err := s.readDirectory(moreIP, subdir); err != nil {
				return err
			}
		}
	}
	return nil
}
