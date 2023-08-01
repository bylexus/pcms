package main

import (
	"embed"
	"flag"
	"fmt"
	"os"
	"path"

	"alexi.ch/pcms/commands"
	"alexi.ch/pcms/model"
)

// embed the site-template/ dir into the binary:
//
//go:embed site-template
var templateContent embed.FS

// embed the built doc folder:
//
//go:embed doc/build
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

* build: Builds the site as static content
* serve: Builds the site (same as build) and starts a webserver to serve the content
* serve-doc: Serves the embedded (binary-built-in) documentation
* init: initializes a directory with a skeleton page
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

	// build command:
	buildCmd := flag.NewFlagSet("build", flag.ExitOnError)
	prevBuildUsage := buildCmd.Usage
	buildCmd.Usage = func() {
		fmt.Fprintf(os.Stderr, "build:      Builds the site to the dest folder\n")
		prevBuildUsage()
		fmt.Fprintln(os.Stderr, "")

	}
	subCommands[buildCmd.Name()] = buildCmd

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
		fmt.Fprintf(os.Stderr, "serve-doc:      Starts a webserver and seves the embedded documentation\n")
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

	// change the app's CWD to the conf file location's dir:
	cwd := path.Dir(confFilePath)
	err := os.Chdir(cwd)
	if err != nil {
		panic(err)
	}
	config := model.NewConfig(confFilePath, args)
	config.EmbeddedDocFS = embeddedDocFS

	switch args.FlagSet.Name() {
	case "build":
		err = commands.RunBuildCmd(config)
	case "serve":
		err = commands.RunServeCmd(config)
	case "serve-doc":
		config.Server.Logging.Access.File = "STDOUT"
		config.Server.Logging.Error.File = "STDERR"
		err = commands.RunServeCmd(config)
	case "init":
		commands.RunInitCmd(args, &templateContent)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
