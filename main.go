package main

import (
	"flag"
	"fmt"
	"os"
)

type CmdArgs struct {
	flagSet *flag.FlagSet
}

func printUsage(flagMap map[string]*flag.FlagSet) {
	fmt.Fprint(os.Stderr, "Usage:\n\npcms [options] <sub-command> [sub-command options]\n\n")
	fmt.Fprint(os.Stderr, "options:\n\n")
	flagMap["__main__"].PrintDefaults()
	delete(flagMap, "__main__")

	fmt.Fprint(os.Stderr, "\nA sub-command is expected. Supported sub-commands:\n\n")
	for _, flagSet := range flagMap {
		flagSet.Usage()
	}
}

func parseCmdArgs() CmdArgs {
	args := CmdArgs{}

	subCommands := make(map[string]*flag.FlagSet)

	helpFlag := flag.Bool("h", false, "Prints this help")
	flag.Parse()

	subCommands["__main__"] = flag.CommandLine

	serveCmd := flag.NewFlagSet("serve", flag.ExitOnError)
	serveCmd.String("c", "", "path to the pcms-config.yaml file. The base dir used is the path of the config file.")
	prevServeUsage := serveCmd.Usage
	serveCmd.Usage = func() {
		fmt.Fprintf(os.Stderr, "serve:      Starts the web server and serves the page\n")
		prevServeUsage()
		fmt.Fprintln(os.Stderr, "")

	}
	subCommands[serveCmd.Name()] = serveCmd

	initCmd := flag.NewFlagSet("init", flag.ExitOnError)
	prevInitUsage := initCmd.Usage
	initCmd.Usage = func() {
		fmt.Fprintf(os.Stderr, "init:      initializes a new pcms project dir using a skeleton\n")
		prevInitUsage()
		fmt.Fprintln(os.Stderr, "init [path]: initializes a new pcms skeleton in the given path, creating it if does not exist")
	}
	subCommands[initCmd.Name()] = initCmd

	passwordCmd := flag.NewFlagSet("password", flag.ExitOnError)
	prevPwUsage := passwordCmd.Usage
	passwordCmd.Usage = func() {
		fmt.Fprintf(os.Stderr, "password:      Creates a new encrypted password to be used in the site.users config\n")
		prevPwUsage()
		fmt.Fprintln(os.Stderr, "password [your-password]")
	}
	subCommands[passwordCmd.Name()] = passwordCmd

	if *helpFlag || flag.CommandLine.NArg() < 1 {
		printUsage(subCommands)
		os.Exit(1)
	}

	if flagSet, defined := subCommands[flag.Args()[0]]; defined == true {
		args.flagSet = flagSet
		flagSet.Parse(flag.Args()[1:])
	} else {
		printUsage(subCommands)
		os.Exit(1)

	}
	return args
}

func main() {
	args := parseCmdArgs()
	switch args.flagSet.Name() {
	case "serve":
		runServeCmd(args)
	case "init":
		runInitCmd(args)
	case "password":
		runPasswordCmd(args)
	}
}
