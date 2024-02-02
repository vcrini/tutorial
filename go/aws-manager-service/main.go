package main

import (
	"flag"
	"log"
)

var (
	_seconds = flag.Int("s", 0, "seconds between requests")
)

func main() {
	flag.Parse()
	args := flag.Args()
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
	default:
		log.Fatalf("Unrecognized command %q. "+
			"Command must be one of: find-version, list-services, show-pipeline, start-pipeline-execution, start-services, stop-services, rollback ...", cmd)
	}
}
