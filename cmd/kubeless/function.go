/*
Copyright (c) 2016-2017 Bitnami

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

package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/Sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/kubeless/kubeless/pkg/minio"
	"github.com/kubeless/kubeless/pkg/spec"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
)

var functionCmd = &cobra.Command{
	Use:   "function SUBCOMMAND",
	Short: "function specific operations",
	Long:  `function command allows user to list, deploy, edit, delete functions running on Kubeless`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	functionCmd.AddCommand(deployCmd)
	functionCmd.AddCommand(deleteCmd)
	functionCmd.AddCommand(listCmd)
	functionCmd.AddCommand(callCmd)
	functionCmd.AddCommand(logsCmd)
	functionCmd.AddCommand(describeCmd)
	functionCmd.AddCommand(updateCmd)
	functionCmd.AddCommand(autoscaleCmd)
}

func getKV(input string) (string, string) {
	var key, value string
	if pos := strings.IndexAny(input, "=:"); pos != -1 {
		key = input[:pos]
		value = input[pos+1:]
	} else {
		// no separator found
		key = input
		value = ""
	}

	return key, value
}

func parseLabel(labels []string) map[string]string {
	funcLabels := map[string]string{}
	for _, label := range labels {
		k, v := getKV(label)
		funcLabels[k] = v
	}
	return funcLabels
}

func parseEnv(envs []string) []v1.EnvVar {
	funcEnv := []v1.EnvVar{}
	for _, env := range envs {
		k, v := getKV(env)
		funcEnv = append(funcEnv, v1.EnvVar{
			Name:  k,
			Value: v,
		})
	}
	return funcEnv
}

func parseMemory(mem string) (resource.Quantity, error) {
	quantity, err := resource.ParseQuantity(mem)
	if err != nil {
		return resource.Quantity{}, err
	}

	return quantity, nil
}

func getFileSha256(file string) (string, error) {
	var checksum string
	h := sha256.New()
	ff, err := os.Open(file)
	if err != nil {
		return checksum, err
	}
	defer ff.Close()
	_, err = io.Copy(h, ff)
	if err != nil {
		return checksum, err
	}
	checksum = hex.EncodeToString(h.Sum(nil))
	return checksum, err
}

func isMinioAvailable(cli kubernetes.Interface) bool {
	_, err := cli.Core().Services("kubeless").Get("minio", metav1.GetOptions{})
	if err != nil {
		return false
	}
	minioPods, err := cli.Core().Pods("kubeless").List(metav1.ListOptions{
		LabelSelector: "kubeless=minio",
	})
	for i := range minioPods.Items {
		if minioPods.Items[i].Status.Phase != "Running" {
			logrus.Warn("Found unhealthy Minio pod, disabling upload")
			return false
		}
	}
	return true
}

func uploadFunction(file string, cli kubernetes.Interface) (string, string, string, error) {
	var function, contentType, checksum string
	stats, err := os.Stat(file)
	if err != nil {
		return "", "", "", err
	}
	if stats.Size() > int64(50*1024*1024) { // TODO: Make the max file size (50 MB) configurable
		err = errors.New("The maximum size of a function is 50MB")
		return "", "", "", err
	}

	if isMinioAvailable(cli) {
		var rawChecksum string
		rawChecksum, err := getFileSha256(file)
		if err != nil {
			return "", "", "", err
		}
		function, err = minio.UploadFunction(file, rawChecksum, cli)
		if err != nil {
			return "", "", "", err
		}
		checksum = "sha256:" + rawChecksum
		contentType = "URL"
		if err != nil {
			return "", "", "", err
		}
	} else {
		// If an object storage service is not available check
		// that the file is not over 1MB to store it as a Custom Resource
		if stats.Size() > int64(1*1024*1024) {
			err = errors.New("Unable to deploy functions over 1MB withouth a storage service")
			return "", "", "", err
		}
		functionBytes, err := ioutil.ReadFile(file)
		if err != nil {
			return "", "", "", err
		}
		if err != nil {
			return "", "", "", err
		}
		fileType := http.DetectContentType(functionBytes)
		if strings.Contains(fileType, "text/plain") {
			function = string(functionBytes[:])
			contentType = "text"
		} else {
			function = base64.StdEncoding.EncodeToString(functionBytes)
			contentType = "base64"
		}
		c, err := getFileSha256(file)
		checksum = "sha256:" + c
		if err != nil {
			return "", "", "", err
		}
	}
	return function, contentType, checksum, nil
}

func getFunctionDescription(funcName, ns, handler, file, deps, runtime, topic, schedule, runtimeImage, mem string, triggerHTTP bool, envs, labels []string, defaultFunction spec.Function, cli kubernetes.Interface) (*spec.Function, error) {

	if handler == "" {
		handler = defaultFunction.Spec.Handler
	}

	var function, checksum, contentType string
	if file == "" {
		file = defaultFunction.Spec.File
		contentType = defaultFunction.Spec.ContentType
		function = defaultFunction.Spec.Function
		checksum = defaultFunction.Spec.Checksum
	} else {
		var err error
		function, contentType, checksum, err = uploadFunction(file, cli)
		if err != nil {
			return &spec.Function{}, err
		}
	}

	if deps == "" {
		deps = defaultFunction.Spec.Deps
	}

	if runtime == "" {
		runtime = defaultFunction.Spec.Runtime
	}

	funcType := ""
	switch {
	case triggerHTTP:
		funcType = "HTTP"
		topic = ""
		schedule = ""
		break
	case schedule != "":
		funcType = "Scheduled"
		topic = ""
		break
	case topic != "":
		funcType = "PubSub"
		schedule = ""
		break
	default:
		funcType = defaultFunction.Spec.Type
		topic = defaultFunction.Spec.Topic
		schedule = defaultFunction.Spec.Schedule
	}

	funcEnv := parseEnv(envs)
	if len(funcEnv) == 0 && len(defaultFunction.Spec.Template.Spec.Containers) != 0 {
		funcEnv = defaultFunction.Spec.Template.Spec.Containers[0].Env
	}

	funcLabels := parseLabel(labels)
	if len(funcLabels) == 0 {
		funcLabels = defaultFunction.Metadata.Labels
	}

	resources := v1.ResourceRequirements{}
	if mem != "" {
		funcMem, err := parseMemory(mem)
		if err != nil {
			err := fmt.Errorf("Wrong format of the memory value: %v", err)
			return &spec.Function{}, err
		}
		resource := map[v1.ResourceName]resource.Quantity{
			v1.ResourceMemory: funcMem,
		}
		resources = v1.ResourceRequirements{
			Limits:   resource,
			Requests: resource,
		}
	} else {
		if len(defaultFunction.Spec.Template.Spec.Containers) != 0 {
			resources = defaultFunction.Spec.Template.Spec.Containers[0].Resources
		}
	}

	if len(runtimeImage) == 0 && len(defaultFunction.Spec.Template.Spec.Containers) != 0 {
		runtimeImage = defaultFunction.Spec.Template.Spec.Containers[0].Image
	}

	return &spec.Function{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Function",
			APIVersion: "k8s.io/v1",
		},
		Metadata: metav1.ObjectMeta{
			Name:      funcName,
			Namespace: ns,
			Labels:    funcLabels,
		},
		Spec: spec.FunctionSpec{
			Handler:     handler,
			Runtime:     runtime,
			Type:        funcType,
			Function:    function,
			File:        path.Base(file),
			Checksum:    checksum,
			ContentType: contentType,
			Deps:        deps,
			Topic:       topic,
			Schedule:    schedule,
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Env:       funcEnv,
							Resources: resources,
							Image:     runtimeImage,
						},
					},
				},
			},
		},
	}, nil
}
