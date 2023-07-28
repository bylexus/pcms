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

func parseCmdArgs() model.CmdArgs {
	args := model.CmdArgs{}

	subCommands := make(map[string]*flag.FlagSet)

	helpFlag := flag.Bool("h", false, "Prints this help")
	confFileFlag := flag.String("c", "pcms-config.yaml", "path to the pcms-config.yaml file. The base dir used is the path of the config file.")
	flag.Parse()

	if confFileFlag != nil {
		args.ConfigFilePath = *confFileFlag
	}

	subCommands["__main__"] = flag.CommandLine

	buildCmd := flag.NewFlagSet("build", flag.ExitOnError)
	prevBuildUsage := buildCmd.Usage
	buildCmd.Usage = func() {
		fmt.Fprintf(os.Stderr, "build:      Builds the site to the dest folder\n")
		prevBuildUsage()
		fmt.Fprintln(os.Stderr, "")

	}
	subCommands[buildCmd.Name()] = buildCmd

	// serveCmd := flag.NewFlagSet("serve", flag.ExitOnError)
	// serveCmd.String("c", "", "path to the pcms-config.yaml file. The base dir used is the path of the config file.")
	// prevServeUsage := serveCmd.Usage
	// serveCmd.Usage = func() {
	// 	fmt.Fprintf(os.Stderr, "serve:      Starts the web server and serves the page\n")
	// 	prevServeUsage()
	// 	fmt.Fprintln(os.Stderr, "")

	// }
	// subCommands[serveCmd.Name()] = serveCmd

	// initCmd := flag.NewFlagSet("init", flag.ExitOnError)
	// prevInitUsage := initCmd.Usage
	// initCmd.Usage = func() {
	// 	fmt.Fprintf(os.Stderr, "init:      initializes a new pcms project dir using a skeleton\n")
	// 	prevInitUsage()
	// 	fmt.Fprintln(os.Stderr, "init [path]: initializes a new pcms skeleton in the given path, creating it if does not exist")
	// 	fmt.Fprintln(os.Stderr, "")
	// }
	// subCommands[initCmd.Name()] = initCmd

	// passwordCmd := flag.NewFlagSet("password", flag.ExitOnError)
	// prevPwUsage := passwordCmd.Usage
	// passwordCmd.Usage = func() {
	// 	fmt.Fprintf(os.Stderr, "password:      Creates a new encrypted password to be used in the site.users config")
	// 	prevPwUsage()
	// 	fmt.Fprintln(os.Stderr, "password [your-password]")
	// 	fmt.Fprintln(os.Stderr, "")
	// }
	// subCommands[passwordCmd.Name()] = passwordCmd

	if *helpFlag || flag.CommandLine.NArg() < 1 {
		printUsage(subCommands)
		os.Exit(1)
	}

	if flagSet, defined := subCommands[flag.Args()[0]]; defined == true {
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
	config := model.NewConfig(confFilePath)
	config.ConfigFile = confFilePath

	switch args.FlagSet.Name() {
	case "build":
		commands.RunBuildCmd(config)
	case "serve":
		// commands.RunServeCmd(args)
	case "init":
		// commands.RunInitCmd(args, &templateContent)
	case "password":
		// commands.RunPasswordCmd(args)
	}
}
