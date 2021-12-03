/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"github.com/spf13/cobra"
	"stash.aspect.com/vopauto/aws-backup/internal"
	"stash.aspect.com/vopauto/aws-backup/internal/protect"
)

var options internal.CommandLineArgs

var protectCmd = &cobra.Command{
	Use:   "protect",
	Short: "Protects specified resources with awsbackup",

	Run: func(cmd *cobra.Command, args []string) {
		protect.Run(options)
	},
}

func init() {
	rootCmd.AddCommand(protectCmd)

	protectCmd.Flags().StringVarP(&options.Filter, "filter", "f", "", "Resource to protect - tag:<tag-name>=<tag-value")
	protectCmd.Flags().StringSliceVarP(&options.ResourceTypes, "resource-types", "r", options.ResourceTypes, "Resource Types - dynamodb,instance,volume,efs,fsx,rds-cluster,rds-db, storage-gateway")
	protectCmd.Flags().BoolVarP(&options.DryRun, "dry-run", "d", false, "Perform Dry Run to show what would have happened.")
	protectCmd.MarkFlagRequired("filter")
	protectCmd.MarkFlagRequired("resource-types")
}
