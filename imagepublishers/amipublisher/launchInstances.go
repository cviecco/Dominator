package amipublisher

import (
	"errors"
	"github.com/Symantec/Dominator/lib/awsutil"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func launchInstances(targets awsutil.TargetList, skipList awsutil.TargetList,
	imageSearchTags, vpcSearchTags, subnetSearchTags,
	securityGroupSearchTags awsutil.Tags, instanceType string,
	sshKeyName string, tags map[string]string, logger log.Logger) error {
	if imageSearchTags["Name"] == "" {
		return errors.New("no image Name search tag")
	}
	resultsChannel := make(chan error, 1)
	numTargets, err := awsutil.ForEachTarget(targets, skipList,
		func(awsService *ec2.EC2, account, region string, logger log.Logger) {
			err := launchInstanceInTarget(awsService,
				imageSearchTags, vpcSearchTags, subnetSearchTags,
				securityGroupSearchTags, instanceType, sshKeyName, tags, logger)
			if err != nil {
				logger.Println(err)
			}
			resultsChannel <- err
		},
		logger)
	// Collect results.
	for i := 0; i < numTargets; i++ {
		e := <-resultsChannel
		if e != nil && err == nil {
			err = e
		}
	}
	return err
}

func launchInstanceInTarget(awsService *ec2.EC2,
	imageSearchTags, vpcSearchTags, subnetSearchTags,
	securityGroupSearchTags awsutil.Tags,
	instanceType string, sshKeyName string, tags map[string]string,
	logger log.Logger) error {
	instances, err := getInstances(awsService, tags["Name"])
	if err != nil {
		return err
	}
	if len(instances) > 0 {
		return nil
	}
	image, err := findImage(awsService, imageSearchTags)
	if err != nil {
		return err
	}
	if image == nil {
		// TODO(rgooch): Create bootstrap image.
		return errors.New("no image found")
	}
	instance, err := launchInstance(awsService, image, vpcSearchTags,
		subnetSearchTags, securityGroupSearchTags, instanceType, sshKeyName)
	if err != nil {
		return err
	}
	instanceId := aws.StringValue(instance.InstanceId)
	logger.Printf("launched: %s\n", instanceId)
	if err := createTags(awsService, instanceId, tags); err != nil {
		return nil
	}
	err = awsService.WaitUntilInstanceRunning(&ec2.DescribeInstancesInput{
		InstanceIds: aws.StringSlice([]string{instanceId}),
	})
	if err != nil {
		return err
	}
	logger.Printf("running: %s\n", instanceId)
	return nil
}
