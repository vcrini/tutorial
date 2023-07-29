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
		fmt.Println(result)
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
	}
	//fmt.Println(*cluster)
}
func readJson(fileName string) ([]byte, error) {
	jsonFile, err := os.Open(fileName)
	fmt.Println(fileName)
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
