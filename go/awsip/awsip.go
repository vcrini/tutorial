package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	buildCommand := "aws ec2 describe-instances --filters \"Name=tag:Name,Values=%s\" --query \"Reservations[].Instances[].{id:InstanceId,name:KeyName,ip:PrivateIpAddress}\""
	command := ""
	host := flag.String("host", "bitgdi-test-ecsnode", "Name of hosts or wildcard")
	flag.Parse()
	switch {
	case *host != "":
		command = fmt.Sprintf(buildCommand, *host)
	}
	fmt.Println(command)
	exe(command)
}

func exe(s string) {
	splitCommand := strings.Split(s, " ")
	cmd := exec.Command(splitCommand[0], splitCommand[1:]...)
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Cannot run:\n%s\n %s", cmd, err)
		os.Exit(1)
	}
}
