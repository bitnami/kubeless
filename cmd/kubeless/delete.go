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
	"github.com/Sirupsen/logrus"
	"github.com/kubeless/kubeless/pkg/utils"
	"github.com/spf13/cobra"
	"k8s.io/client-go/pkg/api"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <function_name>",
	Short: "delete a function from Kubeless",
	Long:  `delete a function from Kubeless`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			logrus.Fatal("Need exactly one argument - function name")
		}
		funcName := args[0]

		ns, err := cmd.Flags().GetString("namespace")
		if err != nil {
			logrus.Fatal(err)
		}
		crdClient, err := utils.GetCDRClientOutOfCluster()
		if err != nil {
			logrus.Fatal(err)
		}

		err = utils.DeleteK8sCustomResource(crdClient, funcName, ns)
		if err != nil {
			logrus.Fatal(err)
		}
	},
}

func init() {
	deleteCmd.Flags().StringP("namespace", "", api.NamespaceDefault, "Specify namespace for the function")
}
