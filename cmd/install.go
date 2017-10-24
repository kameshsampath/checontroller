// Copyright © 2017-present Kamesh Sampath  <kamesh.sampath@hotmail.com>
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
	"path/filepath"

	cheinstall "github.com/kameshsampath/checontroller/che/install"
	"github.com/kameshsampath/checontroller/util"

	cher "github.com/kameshsampath/checontroller/che/refresh"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// installCmd represents the install command
var (
	installCmd = &cobra.Command{
		Use:   "install",
		Short: "Install Che on OpenShift",
		Long:  `Installs and Configure Che on OpenShift`,
		Run:   install,
	}
	newStacksURL    string
	imageTag        string
	refreshStacks   bool
	openshiftFlavor string
)

func init() {
	RootCmd.AddCommand(installCmd)

	installCmd.Flags().BoolVarP(&refreshStacks, "refreshstacks", "r", true, `Refresh the stack to make it OpenShift compatible`)
	installCmd.Flags().StringVarP(&imageTag, "imagetag", "t", "latest", `The Che Image tag to use with che-server image stream,
		 possible values are latest, nightly `)
	installCmd.Flags().StringVarP(&openshiftFlavor, "flavor", "f", "minishift", `OpenShift flavor to use valid values are minishift,ocp`)
	//TODO need to move to configmap
	installCmd.Flags().StringVarP(&newStacksURL, "newStackURL", "n", "https://raw.githubusercontent.com/redhat-developer/rh-che/master/assembly/fabric8-stacks/src/main/resources/stacks.json",
		`The new stacks JSON that will replace default stacks when deploying on OpenShift`)
}

func install(cmd *cobra.Command, args []string) {
	log.Infoln("Starting Che Install on OpenShift")

	var kubeconfig *string

	home := homedir.HomeDir()

	log.Debugf("Home Dir :%s\n", home)

	kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)

	if err != nil {
		log.Fatalf("Unable to build config, %v", err)
	}

	if err != nil {
		log.Fatalf("%s", err)
	}

	var openShiftType cheinstall.OpenShiftType

	if "minishift" == openshiftFlavor {
		openShiftType = "minishift"
	} else if "ocp" == openshiftFlavor {
		openShiftType = "ocp"
	}

	namespace := util.DefaultNamespaceFromConfig(kubeconfig)

	i := cheinstall.NewInstaller(config, namespace, imageTag, openShiftType)

	i.OpenShiftType.Install()

	_, cheEndpointURI := util.CheRouteInfo(config, namespace, "che")

	if refreshStacks {
		log.Infoln("Refreshing Stacking post install")

		log.Infof("Using Che Endpoint URI :%s", cheEndpointURI)
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			log.Fatalf("Unable to build client for refresh %s", err)
		}

		c := cher.NewCheController(cheEndpointURI, namespace, newStacksURL, false, clientset.CoreV1Client.RESTClient())

		cher.TickAndRefresh(c)
	}

	log.Infof("Che is available at: %s", cheEndpointURI)
}
