package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/vcrini/go-utils"
	"os"
	"regexp"
	"time"
)

func main() {
	cluster := flag.String("cluster", "", "cluster where to operate, used with 'list-services', 'stop-services', 'start-services'")
	listServices := flag.Bool("list-services", false, "return services with desired tasks {\"pipeline1\": 1, \"pipeline2\": 0]")
	rollback := flag.String("rollback", "", "service to be rollbacked")
	seconds := flag.Int("s", 0, "seconds between requests")
	showPipeline := flag.String("show-pipeline", "", "show pipeline status")
	startPipelineExecution := flag.String("start-pipeline-execution", "", "given filename containing json with array of pipelines to start: e.g. [\"pipeline1\",\"pipeline2\"] triggers specified pipelines")
	startService := flag.String("start-services", "", "reads a json dictionary where keys are services to start if value is '0' or higher")
	stopService := flag.String("stop-services", "", "reads a json dictionary where keys are services to stop")
	version := flag.Int("version", -1, "version to rollback")
	findVersion := flag.String("findVersion", "", "name of service you want to find version")
	flag.Parse()
	if *startPipelineExecution != "" {
		s := *startPipelineExecution
		byteValue, err := readJson(s)
		if err != nil {
			fmt.Println("could not parse json file")
			os.Exit(1)
		}
		var result []interface{}
		err = json.Unmarshal([]byte(byteValue), &result)
		if err != nil {
			fmt.Printf("can't unmarshall json: %s", err.Error())
			os.Exit(2)
		}
		for _, v := range result {
			fmt.Println(v)
			buildCommand := []string{"aws", "codepipeline", "start-pipeline-execution", "--name", v.(string)}
			fmt.Println(utils.Exe(buildCommand))
			time.Sleep(time.Duration(*seconds) * time.Second)
		}
	} else if *showPipeline != "" {
		s := *showPipeline
		byteValue, err := readJson(s)
		if err != nil {
			fmt.Println("could not parse json file")
			os.Exit(1)
		}
		var result []interface{}
		err = json.Unmarshal([]byte(byteValue), &result)
		if err != nil {
			fmt.Printf("can't unmarshall json: %s", err.Error())
			os.Exit(2)
		}
		for _, v := range result {
			fmt.Println(v)
			buildCommand := []string{"aws", "codepipeline", "list-action-executions", "--pipeline-name", v.(string), "--query", "actionExecutionDetails[0].{status: status, stageName: stageName, startTime: startTime, lastUpdateTime: lastUpdateTime}"}
			fmt.Println(utils.Exe(buildCommand))
			time.Sleep(time.Duration(*seconds) * time.Second)
		}
	} else if *listServices {
		services := make(map[string]int)
		if *cluster == "" {
			fmt.Println("parameter 'cluster' is mandatory")
			os.Exit(4)

		}
		buildCommand := []string{"aws", "ecs", "list-services", "--cluster", *cluster, "--query", "serviceArns[*]"}
		var out = utils.Exe(buildCommand)
		var result []interface{}
		var err = json.Unmarshal([]byte(out), &result)
		if err != nil {
			fmt.Printf("can't unmarshall json: %s", err.Error())
			os.Exit(2)
		}
		r, _ := regexp.Compile("([^/]+)$")
		// to avoid too many requests
		for _, chunk := range chunkBy(result, 10) {
			var names []string
			for _, v := range chunk {
				var name = r.FindString(v.(string))
				names = append(names, name)
			}
			buildCommand := []string{"aws", "ecs", "describe-services", "--cluster", *cluster, "--services"}
			buildCommand = append(buildCommand, names...)
			buildCommand = append(buildCommand, []string{"--query", "services[*].{serviceName: serviceName, desiredCount: desiredCount}"}...)
			out = utils.Exe(buildCommand)
			var result2 []interface{}
			var err = json.Unmarshal([]byte(out), &result2)
			if err != nil {
				fmt.Printf("can't unmarshall json: %s", err.Error())
				os.Exit(2)
			}
			for _, x := range result2 {
				var k = x.(map[string]interface{})["serviceName"].(string)
				var v = int(x.(map[string]interface{})["desiredCount"].(float64))
				services[k] = v
			}
			time.Sleep(time.Duration(*seconds) * time.Second)
		}
		u, err := json.Marshal(services)
		if err != nil {
			fmt.Printf("can't marshall services: %s", err.Error())
			os.Exit(3)
		}
		fmt.Print(string(u))
	} else if *startService != "" {
		startOrStop(*startService, *cluster, "start")
	} else if *stopService != "" {
		startOrStop(*stopService, *cluster, "stop")
	} else if *rollback != "" {
		if *cluster == "" {
			fmt.Println("parameter 'cluster' is mandatory")
			os.Exit(4)
		}
		if *version < 0 {

			fmt.Println("parameter 'version' is mandatory and must be >0")
		}
		deployOrRollback(*rollback, *cluster, *version)
	} else if *findVersion != "" {
		findVersionMax(*findVersion)
	} else {
		fmt.Println("Please use: -h for launch details ")
	}
}
