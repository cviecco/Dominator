package main

import (
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/flagutil"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/lib/srpc/setupclient"
	"os"
)

var (
	networkSpeedPercent = flag.Uint("networkSpeedPercent",
		constants.DefaultNetworkSpeedPercent,
		"Network speed as percentage of capacity")
	scanExcludeList  flagutil.StringList = constants.ScanExcludeList
	scanSpeedPercent                     = flag.Uint("scanSpeedPercent",
		constants.DefaultScanSpeedPercent,
		"Scan speed as percentage of capacity")
	domHostname = flag.String("domHostname", "localhost",
		"Hostname of dominator")
	domPortNum = flag.Uint("domPortNum", constants.DominatorPortNumber,
		"Port number of dominator")
)

func init() {
	flag.Var(&scanExcludeList, "scanExcludeList",
		"Comma separated list of patterns to exclude from scanning")
}

func printUsage() {
	fmt.Fprintln(os.Stderr,
		"Usage: domtool [flags...] disable-updates|enable-updates|configure-subs")
	fmt.Fprintln(os.Stderr, "Common flags:")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  disable-updates reason")
	fmt.Fprintln(os.Stderr, "  enable-updates reason")
	fmt.Fprintln(os.Stderr, "  configure-subs")
}

type commandFunc func(*srpc.Client, []string)

type subcommand struct {
	command string
	numArgs int
	cmdFunc commandFunc
}

var subcommands = []subcommand{
	{"disable-updates", 1, disableUpdatesSubcommand},
	{"enable-updates", 1, enableUpdatesSubcommand},
	{"configure-subs", 0, configureSubsSubcommand},
}

func main() {
	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg() < 1 {
		printUsage()
		os.Exit(2)
	}
	if err := setupclient.SetupTls(true); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	clientName := fmt.Sprintf("%s:%d", *domHostname, *domPortNum)
	client, err := srpc.DialHTTP("tcp", clientName, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error dialing\t%s\n", err)
		os.Exit(1)
	}
	for _, subcommand := range subcommands {
		if flag.Arg(0) == subcommand.command {
			if flag.NArg()-1 != subcommand.numArgs {
				printUsage()
				os.Exit(2)
			}
			subcommand.cmdFunc(client, flag.Args()[1:])
			os.Exit(3)
		}
	}
	printUsage()
	os.Exit(2)
}