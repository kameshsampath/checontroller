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
	"flag"
	cher "github.com/kameshsampath/checontroller/che/refresh"
	"github.com/kameshsampath/checontroller/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
)

// refreshCmd represents the refresh command
var (
	refreshCmd = &cobra.Command{
		Use:   "refresh",
		Short: "Refreshes all the Che stack to make it OpenShift compatible",
		Long:  `Refreshes all the Che stack to make it OpenShift compatible, typically deleteing all the stacks and loading fresh list of compatible stack.`,
		Run:   refresh,
	}
	cheEndpointURI string
	newStackURL    string
	incluster      *bool
)

func init() {
	RootCmd.AddCommand(refreshCmd)

	incluster = refreshCmd.Flags().Bool("incluster", false, "Where the controller will running")

	refreshCmd.Flags().StringVarP(&cheEndpointURI, "endpointURI", "e", "http://localhost:8080", "The Che endpoint URI")
	refreshCmd.Flags().StringVarP(&newStackURL, "newStackURL", "n", "https://raw.githubusercontent.com/redhat-developer/rh-che/master/assembly/fabric8-stacks/src/main/resources/stacks.json", "The JSON from where to load the new stacks")

}

//refresh will handle the Che StackRefreshing calls
func refresh(cmd *cobra.Command, args []string) {

	log.Infof("Che  Endpoint URI %s", cheEndpointURI)
	log.Infof("New Stack URI %s", newStackURL)

	var kubeconfig, podNamespace *string

	var clientset *kubernetes.Clientset

	home := homedir.HomeDir()

	log.Debugf("Home Dir :%s\n", home)

	kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	podNamespace = flag.String("namespace", "", "The Kubernetes Namespace to use")
	flag.Parse()

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
		config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)

		if err != nil {
			log.Fatalf("Unable to build kubeconfig %s", err)
		}
		if *podNamespace == "" {
			*podNamespace = util.DefaultNamespaceFromConfig(kubeconfig)
		}
		//creates clientset
		clientset, err = kubernetes.NewForConfig(config)
		if err != nil {
			log.Fatalf("Unable to build client %s", err)
		}
	}

	log.Infof("Using Namespace: %s", *podNamespace)

	//create controller
	c := cher.NewCheController(cheEndpointURI, *podNamespace, newStackURL,
		*incluster, clientset.CoreV1Client.RESTClient())

	//Daemon mode
	if *incluster {
		cher.KeepAlive(c)
	} else { //Poller mode
		cher.TickAndRefresh(c)
	}

}
