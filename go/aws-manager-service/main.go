package main

import (
	"flag"
	"log"
	"os"
)

var (
	_seconds = flag.Int("s", 0, "seconds between requests")
	_version = flag.Bool("v", false, "return version")
	version  = "0.6.1"
)

func main() {
	flag.Parse()
	args := flag.Args()
	if *_version {
		log.Printf("Version %s", version)
		os.Exit(0)
	}
	if len(args) == 0 {
		log.Fatal("Please specify a subcommand.")
	}
	cmd, args := args[0], args[1:]
	switch cmd {
	case "list-services":
		listServices(args)
	case "rollback":
		rollback(args)
	case "show-pipeline":
		showPipeline(args)
	case "start-services":
		startOrStop("start", args)
	case "stop-services":
		startOrStop("stop", args)
	case "start-pipeline-execution":
		startPipelineExecution(args)
	case "find-version":
		findVersionMax(args)
	case "enable-pipeline":
		EnableOrDisablePipeline("enable", args)
	case "disable-pipeline":
		EnableOrDisablePipeline("disable", args)
	default:
		log.Fatalf("Unrecognized command %q. "+
			"Command must be one of: disable-pipeline, enable-pipeline, find-version, list-services, show-pipeline, start-pipeline-execution, start-services, stop-services, rollback ...", cmd)
	}
}
