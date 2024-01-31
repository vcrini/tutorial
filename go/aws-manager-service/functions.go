package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/vcrini/go-utils"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
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
func findVersionMax(service string) {
	// find oldest version available
	buildCommand := []string{"aws", "ecr", "list-images", "--repository-name", fmt.Sprintf("%s-snapshot", service), "--query", "imageIds[-1].imageTag", "--output", "text"}
	versionOldestSnapshot := strings.TrimSuffix(utils.Exe(buildCommand), "\n")

	buildCommand = []string{"aws", "ecr", "list-images", "--repository-name", service, "--query", "imageIds[-1].imageTag", "--output", "text"}
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
	var err = json.Unmarshal([]byte(out), &result)
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
func startOrStop(file string, cluster string, action string) {
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
