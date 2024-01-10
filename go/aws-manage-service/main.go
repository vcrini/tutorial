package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/vcrini/go-utils"
	"io"
	"os"
	"regexp"
	"time"
)

func main() {
	seconds := flag.Int("s", 0, "seconds between requests")
	cluster := flag.String("cluster", "", "cluster where to operate, used with 'list-services', 'stop-services', 'start-services'")
	startPipelineExecution := flag.String("start-pipeline-execution", "", "given filename containing json with array of pipelines to start: e.g. [\"pipeline1\",\"pipeline2\"] triggers specified pipelines")
	showPipeline := flag.String("show-pipeline", "", "show pipeline status")
	listServices := flag.Bool("list-services", false, "return services with desired tasks {\"pipeline1\": 1, \"pipeline2\": 0]")
	stopService := flag.String("stop-services", "", "reads a json dictionary where keys are services to stop")
	startService := flag.String("start-services", "", "reads a json dictionary where keys are services to start if value is '0' or higher")
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
		start_or_stop(*startService, *cluster, "start")
	} else if *stopService != "" {
		start_or_stop(*stopService, *cluster, "stop")
	} else {
		fmt.Println("Please use: -h for launch details ")
	}
}
func readJson(fileName string) ([]byte, error) {
	jsonFile, err := os.Open(fileName)
	if err != nil {
		return []byte("error"), errors.New("could not parse json file")
	}
	// read our opened jsonFile as a byte array.
	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		return []byte("error"), errors.New("could not parse a byte array")
	}
	defer jsonFile.Close()
	return byteValue, nil
}
func chunkBy[T any](items []T, chunkSize int) (chunks [][]T) {
	for chunkSize < len(items) {
		items, chunks = items[chunkSize:], append(chunks, items[0:chunkSize:chunkSize])
	}
	return append(chunks, items)
}
func start_or_stop(file string, cluster string, action string) {
	byteValue, err := readJson(file)
	if err != nil {
		fmt.Println("could not parse json file")
		os.Exit(1)
	}
	var result map[string]interface{}
	err = json.Unmarshal([]byte(byteValue), &result)
	if err != nil {
		fmt.Printf("can't unmarshall json: %s", err.Error())
		os.Exit(2)
	}
	for k, v := range result {

		var count int
		if action == "stop" {
			count = 0
		} else {
			count = int(v.(float64))
		}
		fmt.Printf("service: %s is being set to: %s\n", k, action)
		buildCommand := []string{"aws", "ecs", "update-service", "--cluster", cluster, "--service", k, "--desired-count", fmt.Sprint(count), "--query", "service.{desiredCount: desiredCount}"}
		fmt.Println(utils.Exe(buildCommand))
	}

}
