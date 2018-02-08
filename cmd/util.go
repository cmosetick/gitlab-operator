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
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"strings"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// Assumes this process is running within a pod in a k8s cluster. Returns a
// config and clientset for the cluster.
func GetInCluster() (*rest.Config, *kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, err
	}

	return config, clientset, nil
}

const NamespaceFilename = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

// Returns the namespace of the pod this process is running within.
func GetNamespace() (string, error) {
	if data, err := ioutil.ReadFile(NamespaceFilename); err != nil {
		return "", err
	} else {
		return string(data), nil
	}
}

// Returns a slice of podNames matching the key=value label.
func GetPodsWithLabel(namespace, key, value string) ([]string, error) {
	_, clientset, err := GetInCluster()
	if err != nil {
		return nil, err
	}

	selector := metav1.LabelSelector{}
	metav1.AddLabelToSelector(&selector, key, value)
	labelSelector := metav1.FormatLabelSelector(&selector)

	pods, err := clientset.Core().Pods(namespace).List(metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return nil, fmt.Errorf("unable to list pods: err %v\n", err)
	}

	var podNames []string
	for _, pod := range pods.Items {
		podNames = append(podNames, pod.Name)
	}

	return podNames, nil
}

// ExecOptions passed to ExecWithOptions
type ExecOptions struct {
	Command []string

	Namespace     string
	PodName       string
	ContainerName string

	Stdin         io.Reader
	CaptureStdout bool
	CaptureStderr bool
	// If false, whitespace in std{err,out} will be removed.
	PreserveWhitespace bool
}

// ExecWithOptions executes a command in the specified container,
// returning stdout, stderr and error. `options` allowed for
// additional parameters to be passed.
func ExecWithOptions(options ExecOptions) error {
	var stdout, stderr bytes.Buffer

	fmt.Printf("Running %v\n", options.Command)

	config, clientset, err := GetInCluster()
	if err != nil {
		return err
	}
	const tty = false

	req := clientset.Core().RESTClient().Post().
		Resource("pods").
		Name(options.PodName).
		Namespace(options.Namespace).
		SubResource("exec").
		Param("container", options.ContainerName)
	req.VersionedParams(&v1.PodExecOptions{
		Container: options.ContainerName,
		Command:   options.Command,
		Stdin:     options.Stdin != nil,
		Stdout:    options.CaptureStdout,
		Stderr:    options.CaptureStderr,
		TTY:       tty,
	}, scheme.ParameterCodec)

	err = execute("POST", req.URL(), config, options.Stdin, &stdout, &stderr, tty)

	if options.PreserveWhitespace {
		fmt.Printf("%v\n%v\n", stdout.String(), stderr.String())
		return err

	}

	fmt.Printf("%v\n%v\n", strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()))
	return err
}

func execute(method string, url *url.URL, config *rest.Config, stdin io.Reader, stdout, stderr io.Writer, tty bool) error {
	exec, err := remotecommand.NewSPDYExecutor(config, method, url)
	if err != nil {
		return err
	}
	return exec.Stream(remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
		Tty:    tty,
	})
}
