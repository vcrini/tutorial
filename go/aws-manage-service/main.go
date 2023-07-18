package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

func main() {
	buildCommand := []string{}
	//cluster := flag.String("cluster", "bitgdi-test-cluster", "cluster name")
	flag.Func("start-pipeline-execution", "Filename containing json with array of pipelines to start: e.g. [\"pipeline1\",\"pipeline2\"]", func(s string) error {
		jsonFile, err := os.Open(s)
		fmt.Println(s)
		if err != nil {
			return errors.New("could not parse json file")
		}
		// read our opened jsonFile as a byte array.
		byteValue, _ := io.ReadAll(jsonFile)
		fmt.Println(byteValue)
		var result []interface{}
		err = json.Unmarshal([]byte(byteValue), &result)
		fmt.Println(result)
		if err != nil {
			return fmt.Errorf("can't unmarshall json: %s", err.Error())
		}
		for _, v := range result {
			fmt.Println(v)
			buildCommand = []string{"aws", "codepipeline", "start-pipeline-execution", "--name", v.(string)}
			fmt.Println(exe(buildCommand))
		}
		defer jsonFile.Close()
		return nil
	})
	flag.Parse()
	//fmt.Println(*cluster)
}

func exe(s []string) string {
	cmd := exec.Command(s[0], s[1:]...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot run:\n%s\n%s\n '%s'", strings.Join(s, "+"), out, err)
		os.Exit(1)
	}
	return string(out)
}
