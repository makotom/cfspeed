package main

import (
	"fmt"
	"time"

	"github.com/spf13/pflag"

	"github.com/makotom/cfspeed/cfspeed"
)

var (
	BuildName       = "\b"
	BuildAnnotation = "git"
)

type CmdOpts struct {
	testIP4            bool
	testIP6            bool
	showVersionAndExit bool
}

func parseFlags() CmdOpts {
	ret := CmdOpts{}

	pflag.BoolVarP(&ret.testIP4, "ip4", "4", false, "Ensure measurements over IPv4")
	pflag.BoolVarP(&ret.testIP6, "ip6", "6", false, "Ensure measurements over IPv6")
	pflag.BoolVar(&ret.showVersionAndExit, "version", false, "Show version information and exit")
	pflag.Parse()

	return ret
}

func printTimestamp() {
	fmt.Println()
	fmt.Printf("At: %s\n", time.Now().Format(time.RFC1123Z))
	fmt.Println()
}

func main() {
	cmdOpts := parseFlags()

	fmt.Printf("cfspeed %s (%s)\n", BuildName, BuildAnnotation)
	if cmdOpts.showVersionAndExit {
		return
	}

	// if none specified, pick up a transport protocol automatically and then exit
	if !cmdOpts.testIP4 && !cmdOpts.testIP6 {
		printTimestamp()
		cfspeed.RunAndPrint("tcp")
		return
	}

	// these options are not mutually exclusive
	if cmdOpts.testIP4 {
		printTimestamp()
		cfspeed.RunAndPrint("tcp4")
	}
	if cmdOpts.testIP6 {
		printTimestamp()
		cfspeed.RunAndPrint("tcp6")
	}
}
