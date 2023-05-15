Utility to retrieve Ip address on AWS EC2 by using right tag.
It can contains wildcards
aws ec2 describe-instances --filters "Name=tag:Name,Values=bitgdi-test-ecsnode" --query "Reservations[].Instances[].InstanceId"
aws ec2 describe-instances --filters "Name=tag:Name,Values=bitgdi-test-ecsnode" --query "Reservations[].Instances[].{id:InstanceId,name:KeyName,ip:PrivateIpAddress}"


