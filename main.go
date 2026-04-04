package main

import (
	"embed"
	"flag"
	"fmt"
	"os"

	"alexi.ch/pcms/commands"
	"alexi.ch/pcms/lib"
	"alexi.ch/pcms/model"
)

// embed the site-template/ dir into the binary:
//
//go:embed site-template
var templateContent embed.FS

// embed the built doc folder:
//
//go:embed doc
var embeddedDocFS embed.FS

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

/*
Parses all command line args and commands, and returns a CmdArgs struct
containing all the parsed commands and flags.

The following global argument are supported:

-h: Prints help, and exit
-c <path> Path to the config yaml file. Defaults to './pcms-config.yaml' if noet set

The following commands are supported:

* serve: Starts a webserver to serve the content
* serve-doc: Serves the embedded (binary-built-in) documentation
* init: initializes a directory with a skeleton page
* index: initializes/updates the local pcms db structure
*/
func parseCmdArgs() model.CmdArgs {
	args := model.CmdArgs{}
	subCommands := make(map[string]*flag.FlagSet)

	// Help flat -h
	helpFlag := flag.Bool("h", false, "Prints this help")
	// config file path -c <path>
	confFileFlag := flag.String("c", "pcms-config.yaml", "path to the pcms-config.yaml file. The base dir used is the path of the config file.")
	flag.Parse()

	if confFileFlag != nil {
		args.ConfigFilePath = *confFileFlag
	}

	subCommands["__main__"] = flag.CommandLine

	// serve command:
	serveCmd := flag.NewFlagSet("serve", flag.ExitOnError)
	serveCmd.String("listen", ":3000", "TCP/IP Listen address, e.g. '-listen :3000' or '-listen 127.0.0.1:8888'")
	prevServeUsage := serveCmd.Usage
	serveCmd.Usage = func() {
		fmt.Fprintf(os.Stderr, "serve:      Starts the web server and serves the page\n")
		prevServeUsage()
		fmt.Fprintln(os.Stderr, "")

	}
	subCommands[serveCmd.Name()] = serveCmd

	// serve command:
	serveDocCmd := flag.NewFlagSet("serve-doc", flag.ExitOnError)
	serveDocCmd.String("listen", ":3000", "TCP/IP Listen address, e.g. '-listen :3000' or '-listen 127.0.0.1:8888'")
	prevServeDocUsage := serveDocCmd.Usage
	serveDocCmd.Usage = func() {
		fmt.Fprintf(os.Stderr, "serve-doc:      Starts a webserver and serves the embedded documentation\n")
		prevServeDocUsage()
		fmt.Fprintln(os.Stderr, "")

	}
	subCommands[serveDocCmd.Name()] = serveDocCmd

	// init command:
	initCmd := flag.NewFlagSet("init", flag.ExitOnError)
	prevInitUsage := initCmd.Usage
	initCmd.Usage = func() {
		fmt.Fprintf(os.Stderr, "init:      initializes a new pcms project dir using a skeleton\n")
		prevInitUsage()
		fmt.Fprintln(os.Stderr, "init [path]: initializes a new pcms skeleton in the given path, creating it if does not exist")
		fmt.Fprintln(os.Stderr, "")
	}
	subCommands[initCmd.Name()] = initCmd

	// index command:
	indexCmd := flag.NewFlagSet("index", flag.ExitOnError)
	prevIndexUsage := indexCmd.Usage
	indexCmd.Usage = func() {
		fmt.Fprintf(os.Stderr, "index:      initializes or updates the local pcms db schema\n")
		prevIndexUsage()
		fmt.Fprintln(os.Stderr, "index: creates or migrates the pcms.db to the current schema version (path configurable via database_path in pcms-config.yaml)")
		fmt.Fprintln(os.Stderr, "")
	}
	subCommands[indexCmd.Name()] = indexCmd

	// cache-clear command:
	cacheClearCmd := flag.NewFlagSet("cache-clear", flag.ExitOnError)
	prevCacheClearUsage := cacheClearCmd.Usage
	cacheClearCmd.Usage = func() {
		fmt.Fprintf(os.Stderr, "cache-clear:      clears the page file cache completely\n")
		prevCacheClearUsage()
		fmt.Fprintln(os.Stderr, "")
	}
	subCommands[cacheClearCmd.Name()] = cacheClearCmd

	if *helpFlag || flag.CommandLine.NArg() < 1 {
		printUsage(subCommands)
		os.Exit(1)
	}

	if flagSet, defined := subCommands[flag.Args()[0]]; defined {
		args.FlagSet = flagSet
		flagSet.Parse(flag.Args()[1:])
	} else {
		printUsage(subCommands)
		os.Exit(1)

	}
	return args
}

func main() {
	args := parseCmdArgs()

	confFilePath := model.GetConfFilePath(args.ConfigFilePath)
	if args.FlagSet.Name() == "serve-doc" {
		confFilePath = "doc/pcms-config.yaml"
	}

	config := model.NewConfig(confFilePath, args, embeddedDocFS)
	lib.SetDBPath(config.DatabasePath)
	var err error

	switch args.FlagSet.Name() {
	case "serve":
		err = commands.RunServeCmd(config)
	case "serve-doc":
		config.Server.Logging.Access.File = "STDOUT"
		config.Server.Logging.Error.File = "STDERR"
		config.ServeMode = model.SERVE_MODE_EMBEDDED_DOC
		lib.SetDBPath(":memory:")
		err = commands.RunServeCmd(config)
	case "init":
		commands.RunInitCmd(args, &templateContent)
	case "index":
		err = commands.RunIndexCmd(config)
	case "cache-clear":
		err = commands.RunCacheClearCmd(config)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
