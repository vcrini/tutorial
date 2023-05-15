aws ec2 describe-instances --filters "Name=tag:Name,Values=bitgdi-test-ecsnode" --query "Reservations[].Instances[].InstanceId"
aws ec2 describe-instances --filters "Name=tag:Name,Values=bitgdi-test-ecsnode" --query "Reservations[].Instances[].{id:InstanceId,name:KeyName,ip:PrivateIpAddress}"


