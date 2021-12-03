package protect

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"stash.aspect.com/vopauto/aws-backup/internal"
)

type EC2DescribeInstancesAPI interface {
	DescribeInstances(input *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error)
}

func Run(args internal.CommandLineArgs) {
	filters := internal.AWSFilter(args.Filter)
	cfg, err := config.LoadDefaultConfig(context.TODO())

	var tasks []string

	if err != nil {
		panic(err)
	}
	stsClient := sts.NewFromConfig(cfg)

	stsOutput, err := stsClient.GetCallerIdentity(context.TODO(), nil)

	if err != nil {
		log.Fatal("Unable to get Account Id")
	}

	accountID := *stsOutput.Account

	if args.DryRun {
		tasks = append(tasks, "Ensure Vault")
	}
	err = internal.EnsureVault(args.Prefix)

	// Create Backup Plan

	if err != nil {
		log.Fatalf("Could not ensure vault %v", err)
	}

	backupPlanID, err := internal.EnsureBackupPlan(args.Prefix)

	if err != nil {
		log.Fatalf("Could not get backup plan id %v", err)
	}
	// Ensure the tags selected at command line satisfy the backup constraints
	ec2Client := ec2.NewFromConfig(cfg)

	instances, err := internal.GetInstances(filters, ec2Client)

	if err != nil {
		log.Fatalf("Unable to get instances %v", err)
	}

	if args.DryRun {
		tasks = append(tasks, "Ensure Vault")
	}
	err = internal.EnsureBackupPlanSelection(args.Prefix, accountID, backupPlanID, args.ResourceTypes)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			log.Printf("Backup plan selection %s already exists", args.Prefix)
		} else {
			log.Fatalf("Could not ensure backup plan selection %v", err)
		}

	}

	for _, v := range instances {
		for _, ebsVolume := range v.BlockDeviceMappings {
			fmt.Println(*ebsVolume.Ebs.VolumeId)
		}
	}

}
