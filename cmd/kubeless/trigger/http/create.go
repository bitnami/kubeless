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

package http

import (
	"github.com/kubeless/kubeless/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	kubelessApi "github.com/kubeless/kubeless/pkg/apis/kubeless/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var createCmd = &cobra.Command{
	Use:   "create <http_trigger_name> FLAG",
	Short: "Create a http trigger",
	Long:  `Create a http trigger`,
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) != 1 {
			logrus.Fatal("Need exactly one argument - http trigger name")
		}
		triggerName := args[0]

		ns, err := cmd.Flags().GetString("namespace")
		if err != nil {
			logrus.Fatal(err)
		}
		if ns == "" {
			ns = utils.GetDefaultNamespace()
		}

		functionName, err := cmd.Flags().GetString("function-name")
		if err != nil {
			logrus.Fatal(err)
		}

		kubelessClient, err := utils.GetKubelessClientOutCluster()
		if err != nil {
			logrus.Fatalf("Can not create out-of-cluster client: %v", err)
		}

		_, err = utils.GetFunctionCustomResource(kubelessClient, functionName, ns)
		if err != nil {
			logrus.Fatalf("Unable to find Function %s in namespace %s. Error %s", functionName, ns, err)
		}

		httpTrigger := kubelessApi.HTTPTrigger{}
		httpTrigger.TypeMeta = metav1.TypeMeta{
			Kind:       "HTTPTrigger",
			APIVersion: "kubeless.io/v1beta1",
		}
		httpTrigger.ObjectMeta = metav1.ObjectMeta{
			Name:      triggerName,
			Namespace: ns,
		}
		httpTrigger.ObjectMeta.Labels = map[string]string{
			"created-by": "kubeless",
		}
		httpTrigger.Spec.FunctionName = functionName

		port, err := cmd.Flags().GetInt32("port")
		if err != nil {
			logrus.Fatal(err)
		}
		if port <= 0 || port > 65535 {
			logrus.Fatalf("Invalid port number %d specified", port)
		}

		err = utils.CreateHTTPTriggerCustomResource(kubelessClient, &httpTrigger)
		if err != nil {
			logrus.Fatalf("Failed to deploy HTTP trigger %s in namespace %s. Error: %s", triggerName, ns, err)
		}
		logrus.Infof("HTTP trigger %s created in namespace %s successfully!", triggerName, ns)
	},
}

func init() {
	createCmd.Flags().StringP("namespace", "", "", "Specify namespace for the function")
	createCmd.Flags().StringP("function-name", "", "", "Name of the function to be associated with trigger")
	createCmd.Flags().Bool("headless", false, "Deploy http-based function without a single service IP and load balancing support from Kubernetes. See: https://kubernetes.io/docs/concepts/services-networking/service/#headless-services")
	createCmd.Flags().Int32("port", 8080, "Deploy http-based function with a custom port")
	createCmd.MarkFlagRequired("function-name")
}
