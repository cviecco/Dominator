package tftpbootd

import (
	"net"
	"sync"
	"time"

	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/pin/tftp"
)

type TftpbootServer struct {
	closeClientTimer       *time.Timer
	imageServerAddress     string
	logger                 log.DebugLogger
	tftpdServer            *tftp.Server
	lock                   sync.Mutex
	filesForIPs            map[string]map[string][]byte
	imageServerClientInUse bool
	imageStreamName        string
	imageServerClientLock  sync.Mutex
	imageServerClient      *srpc.Client
}

func New(imageServerAddress, imageStreamName string,
	logger log.DebugLogger) (*TftpbootServer, error) {
	return newServer(imageServerAddress, imageStreamName, logger)
}

func (s *TftpbootServer) RegisterFiles(ipAddr net.IP, files map[string][]byte) {
	s.registerFiles(ipAddr, files)
}

func (s *TftpbootServer) SetImageStreamName(name string) {
	s.setImageStreamName(name)
}

func (s *TftpbootServer) UnregisterFiles(ipAddr net.IP) {
	s.unregisterFiles(ipAddr)
}
