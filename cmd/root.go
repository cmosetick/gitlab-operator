// Copyright © 2016 Samsung CNCT
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"strings"
)

const (
	GitLabLabelKey          = "app"
	GitLabLabelValue        = "gl-gitlab-ce"
	GitLabContainerName     = "gl-gitlab-ce"
	GitLabRemoteRakeDir     = "/var/opt/gitlab/backups"
	GitLabRemoteEtcFile     = GitLabRemoteRakeDir + "/gitlab-etc.tar.gz"
	GitLabLocalBackupPrefix = "gitlab-backup-"
)

var S3Location string
var ExitCode int

// init the careen config viper instance
var operatorConfig = viper.New()

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "gitlab-operator",
	Short: "Operator for backing up and restoring GitLab",
	Long:  `gitlab-operator is a command line interface for backing up and restoring GitLab CNCT GitLab installations for disaster recovery`,
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(initOperatorConfig)

	RootCmd.SetHelpCommand(helpCmd)

	RootCmd.PersistentFlags().StringVarP(
		&S3Location,
		"s3",
		"s",
		"",
		"s3 bucket or object for backups and restores")
}

// Initializes operatorConfig to use flags or ENV variables (in that order).
func initOperatorConfig() {
	operatorConfig.BindPFlag("s3", RootCmd.Flags().Lookup("s3"))

	operatorConfig.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	operatorConfig.SetEnvPrefix("GITLAB_OPERATOR") // prefix for env vars to configure cluster
	operatorConfig.AutomaticEnv()                  // read in environment variables that match
}

func validateArguments(cmd *cobra.Command, args []string) error {
	s3 := operatorConfig.GetString("s3")
	if s3 == "" {
		ExitCode = 1
		return fmt.Errorf("The -s option is required")
	}

	return nil
}
