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
	cheinstall "github.com/kameshsampath/checontroller/che/install"
	"github.com/kameshsampath/checontroller/util"

	cher "github.com/kameshsampath/checontroller/che/refresh"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
)

// installCmd represents the install command
var (
	cmd             *cobra.Command
	appName         string
	newStacksURL    string
	imageTag        string
	refreshStacks   bool
	openshiftFlavor string
	saveAsTemplate  string
	mavenMirrorURL  string
)

//NewInstallCmd builds a new install command
func NewInstallCmd() *cobra.Command {
	cmd = &cobra.Command{
		Use:   "install",
		Short: "Install Che on OpenShift",
		Long:  `Installs and Configure Che on OpenShift`,
		Run:   install,
	}

	cmd.Flags().StringVarP(&openshiftFlavor, "flavor", "f", "minishift", `OpenShift flavor to use valid values are minishift,ocp`)
	cmd.Flags().StringVarP(&appName, "name", "n", "che", `The name to be used for the Application`)
	cmd.Flags().StringVarP(&imageTag, "imagetag", "t", "latest", `The Che Image tag to use with che-server image stream, possible values are latest, nightly `)
	cmd.Flags().StringVarP(&mavenMirrorURL, "maven-mirror-url", "", "", `The maven mirror url that can be used during build within workspace, e.g http://localnexus/`)
	//TODO need to move to configmap
	cmd.Flags().StringVarP(&newStacksURL, "new-stacks-url", "", DefaultNewStackURL, `The new stacks JSON that will replace default stacks when deploying on OpenShift`)
	cmd.Flags().BoolVarP(&refreshStacks, "refreshstacks", "r", true, `Refresh the stack to make it OpenShift compatible`)
	//TODO handle template name
	cmd.Flags().StringVarP(&saveAsTemplate, "save-as-template", "", "che-server-single-users", `Save the Che install as OpenShift template with given name`)

	return cmd
}

//install
func install(cmd *cobra.Command, args []string) {

	log.Infoln("Starting Che Install on OpenShift")

	var openShiftType cheinstall.OpenShiftType

	if "minishift" == openshiftFlavor {
		openShiftType = "minishift"
	} else if "ocp" == openshiftFlavor {
		openShiftType = "ocp"
	}

	i := cheinstall.InstallerConfig{
		AppName:        appName,
		Config:         Config,
		Namespace:      Namespace,
		OpenShiftType:  openShiftType,
		ImageTag:       imageTag,
		SaveAsTemplate: saveAsTemplate,
		MavenMirrorURL: mavenMirrorURL,
	}

	cheinstall.Installer = &i

	i.OpenShiftType.Install()

	_, cheEndpointURI := util.CheRouteInfo(Config, Namespace, i.AppName)

	if refreshStacks {
		log.Infoln("Refreshing Stacking post install")

		log.Infof("Using Che Endpoint URI :%s", cheEndpointURI)
		clientset, err := kubernetes.NewForConfig(Config)
		if err != nil {
			log.Fatalf("Unable to build client for refresh %s", err)
		}

		c := cher.NewCheController(cheEndpointURI, Namespace, newStacksURL, i.AppName, false, clientset.CoreV1Client.RESTClient())

		cher.TickAndRefresh(c)
	}

	log.Infof("\nChe is available at: %s\n", cheEndpointURI)
}
