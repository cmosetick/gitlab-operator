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
	"time"

	"github.com/spf13/cobra"
)

// Find the one expected pod with the label selector in this namespace and run
// GitLab backup on it. Store the result in an s3 bucket.
func Backup(s3Bucket string) error {
	namespace, err := GetNamespace()
	if err != nil {
		return err
	}

	key, value := GitLabLabelKey, GitLabLabelValue
	podNames, err := GetPodsWithLabel(namespace, key, value)
	if err != nil {
		return err
	} else if len(podNames) != 1 {
		return fmt.Errorf("there were %v pods with label %v=%v when exactly one was expected\n", len(podNames), key, value)
	}

	fmt.Printf("Begining backup of GitLab instance %v, %v\n", namespace, podNames[0])

	options := ExecOptions{
		Command:       nil,
		Namespace:     namespace,
		PodName:       podNames[0],
		ContainerName: GitLabContainerName,
		CaptureStdout: true,
		CaptureStderr: true,
	}

	// Remove the contents of the backup directory to avoid resource
	// exhaustion and simplify identifying the backup we are about
	// to generate.
	options.Command = []string{"rm", "-f", GitLabRemoteRakeDir + "/*"}
	err = ExecWithOptions(options)
	if err != nil {
		return err
	}

	// Run the gitlab rake backup command. It will place a tarball
	// in GitLabRemoteRakeDir
	options.Command = []string{"gitlab-rake", "gitlab:backup:create"}
	err = ExecWithOptions(options)
	if err != nil {
		return err
	}

	// Backup additional GitLab configuration. Place the resulting
	// tarball in the same directory as the rake backup.
	options.Command = []string{"tar", "czf", GitLabRemoteEtcFile, "/etc/gitlab"}
	err = ExecWithOptions(options)
	if err != nil {
		return err
	}

	// Create a tarball of the remote backup dir and save it locally.
	localFilename := GitLabLocalBackupPrefix + time.Now().UTC().Format(time.RFC3339) + ".tar.gz"
	src := fileSpec{PodNamespace: namespace,
		PodName: podNames[0],
		File:    GitLabRemoteRakeDir,
	}
	dest := fileSpec{
		File: localFilename,
	}
	err = CopyFromPod(src, dest)
	if err != nil {
		return err
	}

	err = UploadToS3(s3Bucket, localFilename)
	if err != nil {
		return err
	}

	options.Command = []string{"rm", "-f", localFilename}
	err = ExecWithOptions(options)
	if err != nil {
		return err
	}

	fmt.Printf("Finished backup of GitLab instance\n")

	return nil
}

// backupCmd represents the backup command
var backupCmd = &cobra.Command{
	Use:          "backup [-s bucket]",
	Short:        "Backs up GitLab",
	SilenceUsage: true,
	Long:         `Backs up a GitLab deployment and saves the state to an s3 bucket.`,
	PreRunE:      validateArguments,
	Run: func(cmd *cobra.Command, args []string) {
		s3 := operatorConfig.GetString("s3")
		err := Backup(s3)
		if err != nil {
			ExitCode = 1
			return
		}

		ExitCode = 0
		return
	},
}

func init() {
	RootCmd.AddCommand(backupCmd)
}
