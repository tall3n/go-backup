package internal

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/backup"
	backupTypes "github.com/aws/aws-sdk-go-v2/service/backup/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func AWSFilter(filter string) (filters []types.Filter) {
	splitFilter := strings.Split(filter, "=")
	fmt.Println(splitFilter)
	if len(splitFilter) > 1 {
		filters = append(filters, types.Filter{
			Name:   aws.String(splitFilter[0]),
			Values: []string{splitFilter[1]},
		})
		return
	}
	return
}

func determineScheduleCron(region string) (cron string) {
	switch region {
	case "us-east-1":
		cron = "cron(0 05 * * ? *)"
	case "us-west-2":
		cron = "cron(0 05 * * ? *)"
	case "eu-west-1":
		cron = "cron(0 21 * * ? *)"
	case "eu-west-2":
		cron = "cron(0 21 * * ? *)"
	case "ca-central-1":
		cron = "cron(0 05 * * ? *)"
	default:
		cron = "cron(0 05 * * ? *)"
	}
	return
}

func GetInstances(filters []types.Filter, client *ec2.Client) (instances []types.Instance, err error) {
	input := &ec2.DescribeInstancesInput{
		Filters: filters,
	}

	output, err := client.DescribeInstances(context.TODO(), input)

	if err != nil {
		return
	}

	if len(output.Reservations) == 0 {
		err = errors.New("no instances found")
	}
	for _, reservation := range output.Reservations {
		for _, instance := range reservation.Instances {
			fmt.Println(*instance.InstanceId)
			instances = append(instances, instance)
		}
	}

	return
}

func EnsureVault(prefix string) (err error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return
	}

	// Load AWS Backup Client
	backupClient := backup.NewFromConfig(cfg)
	existingVaults, err := backupClient.ListBackupVaults(context.TODO(), &backup.ListBackupVaultsInput{})

	for _, vault := range existingVaults.BackupVaultList {
		if *vault.BackupVaultName == fmt.Sprintf("%v-vault", prefix) {
			return
		}

	}

	backupVaultInput := &backup.CreateBackupVaultInput{
		BackupVaultName: aws.String(prefix + "-" + "vault"),
	}

	_, err = backupClient.CreateBackupVault(context.TODO(), backupVaultInput)

	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return
	}

	return
}

func GetBackupPlanId(prefix string) (backupPlanId string, err error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())

	if err != nil {
		return
	}

	backupClient := backup.NewFromConfig(cfg)

	backupPlans, err := backupClient.ListBackupPlans(context.TODO(), &backup.ListBackupPlansInput{})
	if err != nil {
		return
	}
	for _, backupPlan := range backupPlans.BackupPlansList {
		if *backupPlan.BackupPlanName == prefix+"-"+"plan" {
			backupPlanId = *backupPlan.BackupPlanId
			return
		}

	}

	return

}

func EnsureBackupPlan(prefix string) (backupPlanID string, err error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())

	if err != nil {
		return
	}

	backupClient := backup.NewFromConfig(cfg)
	// Get existing backup plans only create if not there.
	plans, err := backupClient.ListBackupPlans(context.TODO(), &backup.ListBackupPlansInput{})
	if err != nil {
		return
	}

	for _, plan := range plans.BackupPlansList {
		if *plan.BackupPlanName == fmt.Sprintf("%v-plan", prefix) {
			backupPlanID = *plan.BackupPlanId
			return
		}
	}

	input := &backup.CreateBackupPlanInput{
		BackupPlan: &backupTypes.BackupPlanInput{
			BackupPlanName: aws.String(prefix + "-" + "plan"),
			Rules: []backupTypes.BackupRuleInput{
				{
					RuleName:                aws.String(prefix + "-" + "rule"),
					TargetBackupVaultName:   aws.String(prefix + "-" + "vault"),
					ScheduleExpression:      aws.String(determineScheduleCron(cfg.Region)),
					StartWindowMinutes:      aws.Int64(60),
					CompletionWindowMinutes: aws.Int64(5760),
					Lifecycle: &backupTypes.Lifecycle{
						DeleteAfterDays: aws.Int64(4),
					},
				},
			},
		},
	}

	createPlanOutput, err := backupClient.CreateBackupPlan(context.TODO(), input)

	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return
	}
	backupPlanID = *createPlanOutput.BackupPlanId
	return

}

func EnsureBackupPlanSelection(prefix string, accountID string, planID string, resourceTypes []string) (err error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())

	if err != nil {
		return
	}

	backupClient := backup.NewFromConfig(cfg)

	input := &backup.CreateBackupSelectionInput{
		BackupPlanId: aws.String(planID),
		BackupSelection: &backupTypes.BackupSelection{
			SelectionName: aws.String(prefix + "-" + "selection"),
			IamRoleArn:    aws.String(fmt.Sprintf("arn:aws:iam::%v:role/service-role/AWSBackupDefaultServiceRole", accountID)),
			Resources:     resourceArns(resourceTypes),
			ListOfTags: []backupTypes.Condition{
				{
					ConditionType:  backupTypes.ConditionTypeStringequals,
					ConditionKey:   aws.String("aws:ResourceTag/protected"),
					ConditionValue: aws.String("true"),
				},
			},
		},
	}
	_, err = backupClient.CreateBackupSelection(context.TODO(), input)
	if err != nil {
		return
	}
	return
}

func resourceArns(resourceTypes []string) (resourceArns []string) {
	// If by tag then we will assume the tag will be the limiting factor, the
	for _, v := range resourceTypes {
		switch v {
		case "ebs":
			resourceArns = append(resourceArns, "arn:aws:ec2:*:*:volume/*")
		case "instance":
			resourceArns = append(resourceArns, "arn:aws:ec2:*:*:instance/*")
		case "efs":
			resourceArns = append(resourceArns, "arn:aws:elasticfilesystem:*:*:file-system/*")
		case "fsx":
			resourceArns = append(resourceArns, "arn:aws:fsx:*:*:file-system/*")
		case "rds-db":
			resourceArns = append(resourceArns, "arn:aws:rds:*:*:db:*")
		case "rds-cluster":
			resourceArns = append(resourceArns, "arn:aws:storagegateway:*:*:gateway/*")
		case "storage-gateway":
			resourceArns = append(resourceArns, "arn:aws:storagegateway:*:*:gateway/*")
		}

	}

	return
}

// GetResoruces consolidates resource gathering based on input from the user. If instance/ebs selected then a map instances: volumes: is returned from amazon for parsing
func GetResources(resourceType string) {

}
