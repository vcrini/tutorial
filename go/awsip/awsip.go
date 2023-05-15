package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	buildCommand := "aws ec2 describe-instances --filters Name=tag:Name,Values=%s --query Reservations[].Instances[].{id:InstanceId,name:KeyName,ip:PrivateIpAddress}"
	command := ""
	host := flag.String("host", "bitgdi-test-ecsnode", "Name of hosts or wildcard")
	flag.Parse()
	command = fmt.Sprintf(buildCommand, *host)
	fmt.Println(command)
	fmt.Println(exe(command))
}

func exe(s string) string {
	splitCommand := strings.Split(s, " ")
	cmd := exec.Command(splitCommand[0], splitCommand[1:]...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot run:\n%s\n '%s'", out, err)
		os.Exit(1)
	}
	return string(out)
}
