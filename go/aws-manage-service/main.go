package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/vcrini/go-utils"
	"io"
	"os"
	"time"
)

func main() {
	//cluster := flag.String("cluster", "bitgdi-test-cluster", "cluster name")
	seconds := flag.Int("s", 0, "seconds between requests")
	startPipelineExecution := flag.String("start-pipeline-execution", "", "Filename containing json with array of pipelines to start: e.g. [\"pipeline1\",\"pipeline2\"]")
	showPipeline := flag.String("show-pipeline", "", "Filename containing json with array of pipelines to show: e.g. [\"pipeline1\",\"pipeline2\"]")
	listServices := flag.String("list-services", "", "cluster name")
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
	} else if *listServices != "" {
		s := *listServices
		buildCommand := []string{"aws", "ecs", "list-services", "--cluster", s, "--query", "serviceArns[*]"}
		fmt.Println(utils.Exe(buildCommand))
		time.Sleep(time.Duration(*seconds) * time.Second)
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
