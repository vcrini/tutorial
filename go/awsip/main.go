package main

import (
	"flag"
	"fmt"
	"github.com/vcrini/go-utils"
)

func main() {
	host := flag.String("host", "bitgdi-test-ecsnode", "Name of hosts or wildcard")
	instanceStateName := flag.String("instance-state-name", "running", "The state of the instance (pending | running | shutting-down | terminated | stopping | stopped | all ).")
	flag.Parse()
	var h string
	if *instanceStateName == "all" {
		h = fmt.Sprintf(`[{"Name": "tag:Name","Values":["%s"]}]`, *host)
	} else {
		h = fmt.Sprintf(`[{"Name": "tag:Name","Values":["%s"]},{"Name":"instance-state-name","Values":["%s"]}]`, *host, *instanceStateName)
	}
	buildCommand := []string{"aws", "ec2", "describe-instances", "--filters", h, "--query", "Reservations[].Instances[].{id:InstanceId,name:Tags[?Key == 'Name'].Value | [0],ip:PrivateIpAddress,az:Placement.AvailabilityZone}"}
	fmt.Println(utils.Exe(buildCommand))
}
