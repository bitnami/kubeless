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
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/kubeless/kubeless/pkg/runtime"
	"github.com/kubeless/kubeless/pkg/utils"
	"github.com/spf13/cobra"
	"k8s.io/client-go/pkg/api"
)

var updateCmd = &cobra.Command{
	Use:   "update <function_name> FLAG",
	Short: "update a function on Kubeless",
	Long:  `update a function on Kubeless`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			logrus.Fatal("Need exactly one argument - function name")
		}
		funcName := args[0]

		ns, err := cmd.Flags().GetString("namespace")
		if err != nil {
			logrus.Fatal(err)
		}

		handler, err := cmd.Flags().GetString("handler")
		if err != nil {
			logrus.Fatal(err)
		}

		file, err := cmd.Flags().GetString("from-file")
		if err != nil {
			logrus.Fatal(err)
		}

		runtime, err := cmd.Flags().GetString("runtime")
		if err != nil {
			logrus.Fatal(err)
		}

		triggerHTTP, err := cmd.Flags().GetBool("trigger-http")
		if err != nil {
			logrus.Fatal(err)
		}

		schedule, err := cmd.Flags().GetString("schedule")
		if err != nil {
			logrus.Fatal(err)
		}

		topic, err := cmd.Flags().GetString("trigger-topic")
		if err != nil {
			logrus.Fatal(err)
		}

		labels, err := cmd.Flags().GetStringSlice("label")
		if err != nil {
			logrus.Fatal(err)
		}

		envs, err := cmd.Flags().GetStringArray("env")
		if err != nil {
			logrus.Fatal(err)
		}
		runtimeImage, err := cmd.Flags().GetString("runtime-image")
		if err != nil {
			logrus.Fatal(err)
		}

		mem, err := cmd.Flags().GetString("memory")
		if err != nil {
			logrus.Fatal(err)
		}

		previousFunction, err := utils.GetFunction(funcName, ns)
		if err != nil {
			logrus.Fatal(err)
		}

		cli := utils.GetClientOutOfCluster()
		f, err := getFunctionDescription(funcName, ns, handler, file, "", runtime, topic, schedule, runtimeImage, mem, triggerHTTP, envs, labels, previousFunction, cli)
		if err != nil {
			logrus.Fatal(err)
		}

		crdClient, err := utils.GetCDRClientOutOfCluster()
		if err != nil {
			logrus.Fatal(err)
		}

		err = utils.UpdateK8sCustomResource(crdClient, f)
		if err != nil {
			logrus.Fatal(err)
		}
		logrus.Infof("Function %s submitted for deployment", funcName)
	},
}

func init() {
	updateCmd.Flags().StringP("runtime", "", "", "Specify runtime. Available runtimes are: "+strings.Join(runtime.GetRuntimes(), ", "))
	updateCmd.Flags().StringP("handler", "", "", "Specify handler")
	updateCmd.Flags().StringP("from-file", "", "", "Specify code file")
	updateCmd.Flags().StringP("memory", "", "", "Request amount of memory for the function")
	updateCmd.Flags().StringSliceP("label", "", []string{}, "Specify labels of the function")
	updateCmd.Flags().StringArrayP("env", "", []string{}, "Specify environment variable of the function")
	updateCmd.Flags().StringP("namespace", "", api.NamespaceDefault, "Specify namespace for the function")
	updateCmd.Flags().StringP("trigger-topic", "", "", "Deploy a pubsub function to Kubeless")
	updateCmd.Flags().StringP("schedule", "", "", "Specify schedule in cron format for scheduled function")
	updateCmd.Flags().Bool("trigger-http", false, "Deploy a http-based function to Kubeless")
	updateCmd.Flags().StringP("runtime-image", "", "", "Custom runtime image")
}
