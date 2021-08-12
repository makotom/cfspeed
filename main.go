package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/makotom/cfspeed/cfspeed"
)

var (
	BuildName       = "\b"
	BuildAnnotation = "git"

	printer = log.New(os.Stdout, "", 0)
)

type CmdOpts struct {
	testIP4 bool
	testIP6 bool
}

func printTimestamp() {
	printer.Println()
	printer.Printf("At: %s\n", time.Now().Format(time.RFC1123Z))
	printer.Println()
}

func main() {
	cmdOpts := &CmdOpts{}

	cmd := &cobra.Command{
		Use:          "cfspeed",
		Version:      BuildName,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			printer.Printf(cmd.VersionTemplate())

			// if none specified, pick up a transport protocol automatically and then exit
			if !cmdOpts.testIP4 && !cmdOpts.testIP6 {
				printTimestamp()
				return cfspeed.RunAndPrint(printer, "tcp")
			}

			// these options are not mutually exclusive
			if cmdOpts.testIP4 {
				printTimestamp()
				if err := cfspeed.RunAndPrint(printer, "tcp4"); err != nil {
					return err
				}
			}
			if cmdOpts.testIP6 {
				printTimestamp()
				if err := cfspeed.RunAndPrint(printer, "tcp6"); err != nil {
					return err
				}
			}

			return nil
		},
	}

	flags := cmd.Flags()

	flags.BoolVarP(&cmdOpts.testIP4, "ip4", "4", false, "ensure measurements over IPv4")
	flags.BoolVarP(&cmdOpts.testIP6, "ip6", "6", false, "ensure measurements over IPv6")

	cmd.SetVersionTemplate(fmt.Sprintf("cfspeed %s (%s)\n", BuildName, BuildAnnotation))

	if cmd.Execute() != nil {
		os.Exit(1)
	}
}
