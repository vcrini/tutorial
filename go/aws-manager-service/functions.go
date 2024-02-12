package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/vcrini/go-utils"
	"io"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

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
func deployOrRollback(service string, cluster string, version int) {
	buildCommand := []string{"aws", "ecs", "update-service", "--cluster", cluster, "--service", service, "--task-definition", fmt.Sprintf("%s:%d", service, version), "--query", "service.{taskDefinition: taskDefinition, status: status}"}
	fmt.Println(utils.Exe(buildCommand))

}
func EnableOrDisablePipeline(action string, args []string) {
	flag := flag.NewFlagSet("EnableOrDisablePipeline", flag.ExitOnError)
	registerGlobalFlags(flag)
	err := flag.Parse(args)
	if err != nil {
		log.Fatalf("error in parsing params %s", args)
	}
	// works only if params are before args
	args = flag.Args()
	if action != "enable" && action != "disable" {
		log.Fatal("parameter 'action' must be 'enable' or 'disable' only")
	}
	if len(args) == 0 {
		log.Fatal("you must pass a file containing list of services like [\"service1\", \"service2\"]")
	}
	file := args[0]
	byteValue, err := readJson(file)
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
		buildCommand := []string{"aws", "codepipeline", fmt.Sprintf("%s-stage-transition", action), "--pipeline-name", v.(string), "--stage-name", "Source", "--transition-type", "Outbound"}
		if action == "disable" {
			buildCommand = append(buildCommand, "--reason", "disabled by aws-manager-service")
		}
		fmt.Println(utils.Exe(buildCommand))
		time.Sleep(time.Duration(*_seconds) * time.Second)
	}

}
func findVersionMax(args []string) {
	flag := flag.NewFlagSet("find-version", flag.ExitOnError)
	registerGlobalFlags(flag)
	err := flag.Parse(args)
	if err != nil {
		log.Fatalf("error in parsing params %s", args)
	}
	if len(args) == 0 {
		log.Fatal(" you must provide a filename containing json with array of pipelines to start: e.g. [\"pipeline1\",\"pipeline2\"] triggers specified pipelines")
	}
	service := args[0]
	// find oldest version available

	buildCommand := []string{"aws", "ecr", "list-images", "--repository-name", fmt.Sprintf("%s-snapshot", service), "--query", "imageIds[?imageTag!=``].imageTag|[0]", "--output", "text"}
	versionOldestSnapshot := strings.TrimSuffix(utils.Exe(buildCommand), "\n")

	buildCommand = []string{"aws", "ecr", "list-images", "--repository-name", service, "--query", "imageIds[?imageTag!=``].imageTag|[0]", "--output", "text"}
	versionOldestNonSnapshot := strings.TrimSuffix(utils.Exe(buildCommand), "\n")
	var versionList []string
	versionList = append(versionList, versionOldestNonSnapshot, versionOldestSnapshot)
	sort.Strings(versionList)
	versionOldest := versionList[0]

	buildCommand = []string{"aws", "ecs", "list-task-definitions", "--family-prefix", service, "--query", "reverse(taskDefinitionArns[*])"}
	var out = utils.Exe(buildCommand)
	// [
	//     "arn:aws:ecs:eu-west-1:796341525871:task-definition/dpl-app-appdemo-backend:28",
	//     "arn:aws:ecs:eu-west-1:796341525871:task-definition/dpl-app-appdemo-backend:2",
	//     "arn:aws:ecs:eu-west-1:796341525871:task-definition/dpl-app-appdemo-backend:1"
	//
	var result []interface{}
	err = json.Unmarshal([]byte(out), &result)
	if err != nil {
		fmt.Printf("can't unmarshall json: %s", err.Error())
		os.Exit(2)
	}
	rImageVersion := regexp.MustCompile(`([^:\\]+)"`)
	rTaskDefinitionAndVesion, _ := regexp.Compile("([^/]+)$")
	version := ""
	old_version := ""
	versions := make(map[string]string)
	for _, arn := range result {
		var taskDefinitionAndVersion = rTaskDefinitionAndVesion.FindString(arn.(string))
		buildCommand := []string{"aws", "ecs", "describe-task-definition", "--task-definition", taskDefinitionAndVersion, "--query", "taskDefinition.containerDefinitions[0].image"}
		v := utils.Exe(buildCommand)
		version = rImageVersion.FindStringSubmatch(v)[1]
		if version != old_version {
			versions[taskDefinitionAndVersion] = version
		}
		if version == versionOldest {
			// this is oldest version available quit loop
			break
		}
		old_version = version
	}
	u, err := json.Marshal(versions)
	if err != nil {
		fmt.Printf("can't marshall versions: %s", err.Error())
		os.Exit(3)
	}
	fmt.Print(string(u))

}

func listServices(args []string) {
	flag := flag.NewFlagSet("list-services", flag.ExitOnError)
	var (
		cluster = flag.String("cluster", "", "cluster where to operate")
	)
	registerGlobalFlags(flag)
	err := flag.Parse(args)
	if err != nil {
		log.Fatalf("error in parsing params %s", args)
	}
	services := make(map[string]int)
	if *cluster == "" {
		log.Fatal("parameter 'cluster' is mandatory")
	}
	buildCommand := []string{"aws", "ecs", "list-services", "--cluster", *cluster, "--query", "serviceArns[*]"}
	var out = utils.Exe(buildCommand)
	var result []interface{}
	err = json.Unmarshal([]byte(out), &result)
	if err != nil {
		log.Fatalf("can't unmarshall json: %s", err.Error())
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
			log.Fatalf("can't unmarshall json: %s", err.Error())
		}
		for _, x := range result2 {
			var k = x.(map[string]interface{})["serviceName"].(string)
			var v = int(x.(map[string]interface{})["desiredCount"].(float64))
			services[k] = v
		}
		time.Sleep(time.Duration(*_seconds) * time.Second)
	}
	u, err := json.Marshal(services)
	if err != nil {
		log.Fatalf("can't marshall services: %s", err.Error())
	}
	fmt.Print(string(u))
}
func showPipeline(args []string) {
	flag := flag.NewFlagSet("show-pipeline", flag.ExitOnError)
	registerGlobalFlags(flag)
	err := flag.Parse(args)
	if err != nil {
		log.Fatalf("error in parsing params %s", args)
	}
	if len(args) == 0 {
		log.Fatal(" you must provide as show-pipeline argument a file with a json list of pipelines: e.g. [\"pipeline1\",\"pipeline2\"]")
	}
	s := args[0]
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
		time.Sleep(time.Duration(*_seconds) * time.Second)
	}
}
func startPipelineExecution(args []string) {
	flag := flag.NewFlagSet("start-pipeline-execution", flag.ExitOnError)
	registerGlobalFlags(flag)
	err := flag.Parse(args)
	if err != nil {
		log.Fatalf("error in parsing params %s", args)
	}
	// works only if params are before args
	args = flag.Args()
	if len(args) == 0 {
		log.Fatal(" you must provide a filename containing json with array of pipelines to start: e.g. [\"pipeline1\",\"pipeline2\"] triggers specified pipelines")
	}
	s := args[0]
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
		time.Sleep(time.Duration(*_seconds) * time.Second)
	}
}
func startOrStop(action string, args []string) {
	flag := flag.NewFlagSet("startOrStop", flag.ExitOnError)
	var (
		cluster = flag.String("cluster", "", "cluster where to operate")
	)
	registerGlobalFlags(flag)
	err := flag.Parse(args)
	if err != nil {
		log.Fatalf("error in parsing params %s", args)
	}
	// works only if params are before args
	args = flag.Args()
	if *cluster == "" {
		log.Fatal("parameter 'cluster' is mandatory")
	}
	if len(args) == 0 {
		if action == "start" {
			log.Fatal("you must provide as an argument a file with a json dictionary where keys are services to start if value is '0' or higher")
		} else {
			log.Fatal("you must provide as an argument a  file with a json dictionary where keys are services to where keys are services to stop")
		}
	}
	file := args[0]
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
		buildCommand := []string{"aws", "ecs", "update-service", "--cluster", *cluster, "--service", k, "--desired-count", fmt.Sprint(count), "--query", "service.{desiredCount: desiredCount}"}
		fmt.Println(utils.Exe(buildCommand))
	}

}
func rollback(args []string) {
	flag := flag.NewFlagSet("rollback", flag.ExitOnError)
	var (
		cluster = flag.String("cluster", "", "cluster where to operate")
		version = flag.Int("version", -1, "version to rollback")
		service = flag.String("service", "", "service to be rollbacked")
	)
	registerGlobalFlags(flag)
	err := flag.Parse(args)
	if err != nil {
		log.Fatalf("error in parsing params %s", args)
	}
	//args = flag.Args()
	if *service == "" {
		log.Fatal("parameter 'service' is mandatory")
	}
	if *cluster == "" {
		log.Fatal("parameter 'cluster' is mandatory")
	}
	if *version < 0 {

		fmt.Println("parameter 'version' is mandatory and must be >0")
	}
	deployOrRollback(*service, *cluster, *version)
}
func registerGlobalFlags(fset *flag.FlagSet) {
	flag.VisitAll(func(f *flag.Flag) {
		fset.Var(f.Value, f.Name, f.Usage)
	})
}
