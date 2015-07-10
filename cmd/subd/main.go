package main

import (
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/lib/fsbench"
	"github.com/Symantec/Dominator/lib/memstats"
	"github.com/Symantec/Dominator/sub/fsrateio"
	"github.com/Symantec/Dominator/sub/httpd"
	"github.com/Symantec/Dominator/sub/scanner"
	"os"
	"path"
	"runtime"
	"strconv"
	"syscall"
)

var (
	portNum = flag.Uint("portNum", 6969,
		"Port number to allocate and listen on for HTTP/RPC")
	rootDir = flag.String("rootDir", "/",
		"Name of root of directory tree to manage")
	showStats = flag.Bool("showStats", false,
		"If true, show statistics after each cycle")
	subdDir = flag.String("subdDir", "/.subd",
		"Name of subd private directory. This must be on the same file-system as rootdir")
	unshare = flag.Bool("unshare", true, "Internal use only.")
)

func sanityCheck() bool {
	r_devnum, err := fsbench.GetDevnumForFile(*rootDir)
	if err != nil {
		fmt.Printf("Unable to get device number for: %s\t%s\n", *rootDir, err)
		return false
	}
	s_devnum, err := fsbench.GetDevnumForFile(*subdDir)
	if err != nil {
		fmt.Printf("Unable to get device number for: %s\t%s\n", *subdDir, err)
		return false
	}
	if r_devnum != s_devnum {
		fmt.Printf("rootDir and subdDir must be on the same file-system\n")
		return false
	}
	return true
}

func createDirectory(dirname string) bool {
	err := os.MkdirAll(dirname, 0750)
	if err != nil {
		fmt.Printf("Unable to create directory: %s\t%s\n", dirname, err)
		return false
	}
	return true
}

func mountTmpfs(dirname string) bool {
	var statfs syscall.Statfs_t
	err := syscall.Statfs(dirname, &statfs)
	if err != nil {
		fmt.Printf("Unable to create Statfs: %s\t%s\n", dirname, err)
		return false
	}
	if statfs.Type != 0x01021994 {
		err := syscall.Mount("none", dirname, "tmpfs", 0,
			"size=65536,mode=0750")
		if err == nil {
			fmt.Printf("Mounted tmpfs on: %s\n", dirname)
		} else {
			fmt.Printf("Unable to mount tmpfs on: %s\t%s\n", dirname, err)
			return false
		}
	}
	return true
}

func unshareAndBind(workingRootDir string) bool {
	if *unshare {
		// Re-exec myself using the unshare syscall while on a locked thread.
		// This hack is required because syscall.Unshare() operates on only one
		// thread in the process, and Go switches execution between threads
		// randomly. Thus, the namespace can be suddenly switched for running
		// code. This is an aspect of Go that was not well thought out.
		runtime.LockOSThread()
		err := syscall.Unshare(syscall.CLONE_NEWNS)
		if err != nil {
			fmt.Printf("Unable to unshare mount namesace\t%s\n", err)
			return false
		}
		args := append(os.Args, "-unshare=false")
		err = syscall.Exec(args[0], args, os.Environ())
		if err != nil {
			fmt.Printf("Unable to Exec:%s\t%s\n", args[0], err)
			return false
		}
	}
	// Strip out the "-unshare=false" just in case.
	os.Args = os.Args[0 : len(os.Args)-1]
	err := syscall.Mount("none", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, "")
	if err != nil {
		fmt.Printf("Unable to set mount sharing to private\t%s\n", err)
		return false
	}
	err = syscall.Mount(*rootDir, workingRootDir, "", syscall.MS_BIND, "")
	if err != nil {
		fmt.Printf("Unable to bind mount %s to %s\t%s\n",
			*rootDir, workingRootDir, err)
		return false
	}
	return true
}

func getCachedSpeed(workingRootDir string, cacheDirname string) (bytesPerSecond,
	blocksPerSecond uint64, ok bool) {
	bytesPerSecond = 0
	blocksPerSecond = 0
	devnum, err := fsbench.GetDevnumForFile(workingRootDir)
	if err != nil {
		fmt.Printf("Unable to get device number for: %s\t%s\n",
			workingRootDir, err)
		return 0, 0, false
	}
	fsbenchDir := path.Join(cacheDirname, "fsbench")
	if !createDirectory(fsbenchDir) {
		return 0, 0, false
	}
	cacheFilename := path.Join(fsbenchDir, strconv.FormatUint(devnum, 16))
	file, err := os.Open(cacheFilename)
	if err == nil {
		n, err := fmt.Fscanf(file, "%d %d", &bytesPerSecond, &blocksPerSecond)
		file.Close()
		if n == 2 || err == nil {
			return bytesPerSecond, blocksPerSecond, true
		}
	}
	bytesPerSecond, blocksPerSecond, err = fsbench.GetReadSpeed(workingRootDir)
	if err != nil {
		fmt.Printf("Unable to measure read speed\t%s\n", err)
		return 0, 0, false
	}
	file, err = os.Create(cacheFilename)
	if err != nil {
		fmt.Printf("Unable to open: %s for write\t%s\n", cacheFilename, err)
		return 0, 0, false
	}
	fmt.Fprintf(file, "%d %d\n", bytesPerSecond, blocksPerSecond)
	file.Close()
	return bytesPerSecond, blocksPerSecond, true
}

func main() {
	flag.Parse()
	workingRootDir := path.Join(*subdDir, "root")
	objectsDir := path.Join(*subdDir, "objects")
	tmpDir := path.Join(*subdDir, "tmp")
	if !createDirectory(workingRootDir) {
		os.Exit(1)
	}
	if !sanityCheck() {
		os.Exit(1)
	}
	if !createDirectory(objectsDir) {
		os.Exit(1)
	}
	if !createDirectory(tmpDir) {
		os.Exit(1)
	}
	if !mountTmpfs(tmpDir) {
		os.Exit(1)
	}
	if !unshareAndBind(workingRootDir) {
		os.Exit(1)
	}
	bytesPerSecond, blocksPerSecond, ok := getCachedSpeed(workingRootDir,
		tmpDir)
	if !ok {
		os.Exit(1)
	}
	ctx := fsrateio.NewContext(bytesPerSecond, blocksPerSecond)
	fmt.Println(ctx)
	var fsh scanner.FileSystemHistory
	fsChannel := scanner.StartScannerDaemon(workingRootDir, objectsDir, ctx)
	err := httpd.StartServer(*portNum, &fsh)
	if err != nil {
		fmt.Printf("Unable to create http server\t%s\n", err)
		os.Exit(1)
	}
	fsh.Update(nil)
	for iter := 0; true; iter++ {
		fmt.Printf("Starting cycle: %d\n", iter)
		fsh.Update(<-fsChannel)
		runtime.GC() // An opportune time to take out the garbage.
		if *showStats {
			fmt.Print(fsh)
			fmt.Print(fsh.FileSystem())
			memstats.WriteMemoryStats(os.Stdout)
		}
	}
}