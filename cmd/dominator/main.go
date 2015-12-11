package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/dom/herd"
	"github.com/Symantec/Dominator/dom/mdb"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/logbuf"
	"log"
	"os"
	"os/user"
	"path"
	"runtime"
	"strconv"
	"syscall"
	"time"
)

var (
	caFile = flag.String("CAfile", "/etc/ssl/CA.pem",
		"Name of file containing the root of trust")
	certFile = flag.String("certFile", "/etc/ssl/Dominator/cert.pem",
		"Name of file containing the SSL certificate")
	debug = flag.Bool("debug", false,
		"If true, show debugging output")
	fdLimit = flag.Uint64("fdLimit", getFdLimit(),
		"Maximum number of open file descriptors (this limits concurrent connection attempts)")
	imageServerHostname = flag.String("imageServerHostname", "localhost",
		"Hostname of image server")
	imageServerPortNum = flag.Uint("imageServerPortNum",
		constants.ImageServerPortNumber,
		"Port number of image server")
	keyFile = flag.String("keyFile", "/etc/ssl/Dominator/key.pem",
		"Name of file containing the SSL certificate")
	logbufLines = flag.Uint("logbufLines", 1024,
		"Number of lines to store in the log buffer")
	minInterval = flag.Uint("minInterval", 1,
		"Minimum interval between loops (in seconds)")
	portNum = flag.Uint("portNum", constants.DomPortNumber,
		"Port number to allocate and listen on for HTTP/RPC")
	stateDir = flag.String("stateDir", "/var/lib/Dominator",
		"Name of dominator state directory.")
	username = flag.String("username", "",
		"If running as root, username to switch to.")
)

func showMdb(mdb *mdb.Mdb) {
	fmt.Println()
	mdb.DebugWrite(os.Stdout)
	fmt.Println()
}

func getFdLimit() uint64 {
	var rlim syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlim); err != nil {
		panic(err)
	}
	return rlim.Max
}

func setUser(username string) error {
	if username == "" {
		return errors.New("-username argument missing")
	}
	newUser, err := user.Lookup(username)
	if err != nil {
		return err
	}
	uid, err := strconv.Atoi(newUser.Uid)
	if err != nil {
		return err
	}
	gid, err := strconv.Atoi(newUser.Gid)
	if err != nil {
		return err
	}
	if uid == 0 {
		return errors.New("Do not run the Dominator as root")
		os.Exit(1)
	}
	if err := syscall.Setresgid(gid, gid, gid); err != nil {
		return err
	}
	return syscall.Setresuid(uid, uid, uid)
}

func main() {
	flag.Parse()
	setupTls()
	rlim := syscall.Rlimit{*fdLimit, *fdLimit}
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rlim); err != nil {
		fmt.Fprintf(os.Stderr, "Cannot set FD limit\t%s\n", err)
		os.Exit(1)
	}
	if os.Geteuid() == 0 {
		if err := setUser(*username); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	fi, err := os.Lstat(*stateDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot stat: %s\t%s\n", *stateDir, err)
		os.Exit(1)
	}
	if !fi.IsDir() {
		fmt.Fprintf(os.Stderr, "%s is not a directory\n", *stateDir)
		os.Exit(1)
	}
	interval := time.Duration(*minInterval) * time.Second
	circularBuffer := logbuf.New(*logbufLines)
	logger := log.New(circularBuffer, "", log.LstdFlags)
	mdbChannel := mdb.StartMdbDaemon(path.Join(*stateDir, "mdb"), logger)
	herd := herd.NewHerd(fmt.Sprintf("%s:%d", *imageServerHostname,
		*imageServerPortNum), logger)
	herd.AddHtmlWriter(circularBuffer)
	if err = herd.StartServer(*portNum, true); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create http server\t%s\n", err)
		os.Exit(1)
	}
	nextCycleStopTime := time.Now().Add(interval)
	for {
		select {
		case mdb := <-mdbChannel:
			herd.MdbUpdate(mdb)
			if *debug {
				showMdb(mdb)
			}
			runtime.GC() // An opportune time to take out the garbage.
		default:
			// Do work.
			if herd.PollNextSub() {
				if *debug {
					fmt.Print(".")
				}
				sleepTime := nextCycleStopTime.Sub(time.Now())
				time.Sleep(sleepTime)
				nextCycleStopTime = time.Now().Add(interval)
				if sleepTime < 0 { // There was no time to rest.
					runtime.GC() // An opportune time to take out the garbage.
				}
			}
		}
	}
}
