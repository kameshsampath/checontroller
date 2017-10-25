// Copyright © 2017 Kamesh Sampath <kamesh.sampath@hotmail.com>
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
	"os"

	cher "github.com/kameshsampath/checontroller/che/refresh"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// refreshCmd represents the refresh command
var (
	cheEndpointURI  string
	newStackURL     string
	applicationName string
	incluster       *bool
)

//NewRefreshCmd builds new refresh command to refresh che stacks
func NewRefreshCmd() *cobra.Command {

	cmd = &cobra.Command{
		Use:   "refresh",
		Short: "Refreshes all the Che stack to make it OpenShift compatible",
		Long:  `Refreshes all the Che stack to make it OpenShift compatible, typically deleteing all the stacks and loading fresh list of compatible stack.`,
		Run:   refresh,
	}

	incluster = cmd.Flags().Bool("incluster", false, "Where the controller will running, ability to deploy this app as a pod")

	cmd.Flags().StringVarP(&cheEndpointURI, "endpointURI", "e", "http://localhost:8080", "The Che endpoint URI")
	cmd.Flags().StringVarP(&newStackURL, "new-stack-url", "", DefaultNewStackURL, "The JSON from where to load the new stacks")
	cmd.Flags().StringVarP(&applicationName, "application-name", "n", "", "The Che application name, which was used when installing")

	return cmd
}

//refresh will handle the Che StackRefreshing calls
func refresh(cmd *cobra.Command, args []string) {

	log.Infof("Che  Endpoint URI %s", cheEndpointURI)
	log.Infof("New Stack URI %s", newStackURL)

	var podNamespace *string
	var clientset *kubernetes.Clientset
	var err error

	if *incluster {
		log.Infoln("Accessing from inside cluster")
		config, err := rest.InClusterConfig()
		if err != nil {
			log.Fatalf("Unable to build kubeconfig %s", err)
		}
		clientset, err = kubernetes.NewForConfig(config)
		if err != nil {
			log.Fatalf("Unable to build client %s", err)
		}
		*podNamespace = os.Getenv("KUBERNETES_NAMESPACE")
	} else {
		log.Infoln("Accessing from outside cluster")
		*podNamespace = Namespace
		//creates clientset
		clientset, err = kubernetes.NewForConfig(Config)
		if err != nil {
			log.Fatalf("Unable to build client %s", err)
		}
	}

	log.Infof("Using Namespace: %s", *podNamespace)

	//create controller
	c := cher.NewCheController(cheEndpointURI, *podNamespace, newStackURL, applicationName,
		*incluster, clientset.CoreV1Client.RESTClient())

	//Daemon mode
	if *incluster {
		cher.KeepAlive(c)
	} else { //Poller mode
		cher.TickAndRefresh(c)
	}

}
