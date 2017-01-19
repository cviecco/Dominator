package main

import (
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/lib/awsutil"
	"github.com/Symantec/Dominator/lib/constants"
	liblog "github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc/setupclient"
	"log"
	"os"
	"time"
)

var (
	amiName   = flag.String("amiName", "", "AMI Name property")
	expiresIn = flag.Duration("expiresIn", time.Hour,
		"Date to set for the ExpiresAt tag")
	ignoreMissingUnpackers = flag.Bool("ignoreMissingUnpackers", false,
		"If true, do not generate an error for missing unpackers")
	imageServerHostname = flag.String("imageServerHostname", "localhost",
		"Hostname of imageserver")
	imageServerPortNum = flag.Uint("imageServerPortNum",
		constants.ImageServerPortNumber, "Port number of imageserver")
	maxIdleTime = flag.Duration("maxIdleTime", time.Minute*50,
		"Maximum idle time for image unpacker instances")
	minFreeBytes = flag.Uint64("minFreeBytes", 1<<28,
		"minimum number of free bytes in image")
	skipTargets awsutil.TargetList
	tagsFile    = flag.String("tagsFile", "",
		"JSON encoded file containing tags to apply to AMIs")
	tagKey       = flag.String("tagKey", "", "Tag key name to apply")
	tagValue     = flag.String("tagValue", "", "Tag value to apply")
	targets      awsutil.TargetList
	unpackerName = flag.String("unpackerName", "ImageUnpacker",
		"The Name tag value for image unpacker instances")
)

func init() {
	flag.Var(&skipTargets, "skipTargets",
		"List of targets to skip (default none). No wildcards permitted")
	flag.Var(&targets, "targets",
		"List of targets (default all accounts and regions)")
}

func printUsage() {
	fmt.Fprintln(os.Stderr,
		"Usage: ami-publisher [flags...] publish [args...]")
	fmt.Fprintln(os.Stderr, "Common flags:")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  delete results-file...")
	fmt.Fprintln(os.Stderr, "  delete-tags tag-key results-file...")
	fmt.Fprintln(os.Stderr, "  expire")
	fmt.Fprintln(os.Stderr, "  list-unpackers")
	fmt.Fprintln(os.Stderr, "  prepare-unpackers stream-name")
	fmt.Fprintln(os.Stderr, "  publish stream-name image-leaf-name")
	fmt.Fprintln(os.Stderr, "  set-exclusive-tags key value results-file...")
	fmt.Fprintln(os.Stderr, "  set-tags-on-unpackers")
	fmt.Fprintln(os.Stderr, "  stop-idle-unpackers")
}

type commandFunc func([]string, liblog.Logger)

type subcommand struct {
	command string
	minArgs int
	maxArgs int
	cmdFunc commandFunc
}

var subcommands = []subcommand{
	{"delete", 1, -1, deleteSubcommand},
	{"delete-tags", 2, -1, deleteTagsSubcommand},
	{"expire", 0, 0, expireSubcommand},
	{"list-unpackers", 0, 0, listUnpackersSubcommand},
	{"prepare-unpackers", 1, 1, prepareUnpackersSubcommand},
	{"publish", 2, 2, publishSubcommand},
	{"set-exclusive-tags", 2, -1, setExclusiveTagsSubcommand},
	{"set-tags-on-unpackers", 0, 0, setTagsSubcommand},
	{"stop-idle-unpackers", 0, 0, stopIdleUnpackersSubcommand},
}

func main() {
	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg() < 1 {
		printUsage()
		os.Exit(2)
	}
	logger := log.New(os.Stderr, "", log.LstdFlags)
	if err := setupclient.SetupTls(true); err != nil {
		logger.Println(err)
		os.Exit(1)
	}
	numSubcommandArgs := flag.NArg() - 1
	for _, subcommand := range subcommands {
		if flag.Arg(0) == subcommand.command {
			if numSubcommandArgs < subcommand.minArgs ||
				(subcommand.maxArgs >= 0 &&
					numSubcommandArgs > subcommand.maxArgs) {
				printUsage()
				os.Exit(2)
			}
			subcommand.cmdFunc(flag.Args()[1:], logger)
			os.Exit(3)
		}
	}
	printUsage()
	os.Exit(2)
}
