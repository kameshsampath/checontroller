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
	newStacksURL  string
	refreshStacks bool
	minishift     bool
	ocp           bool
	osio          bool
)

func init() {
	RootCmd.AddCommand(installCmd)

	installCmd.Flags().BoolVarP(&refreshStacks, "refreshstacks", "r", true, "Refresh the stack to make it OpenShift compatible")
	installCmd.Flags().BoolVarP(&minishift, "minishift", "m", false, "Is OpenShift cluster running on minishift")
	installCmd.Flags().BoolVarP(&osio, "osio", "o", false, "Is OpenShift cluster running on openshift.io")
	installCmd.Flags().BoolVarP(&ocp, "ocp", "d", false, "Is OpenShift cluster running on OpenShift Container Platform")
	//TODO need to move to configmap
	installCmd.Flags().StringVarP(&newStacksURL, "newStackURL", "n", "https://raw.githubusercontent.com/redhat-developer/rh-che/master/assembly/fabric8-stacks/src/main/resources/stacks.json",
		"The JSON from where to load the new stacks during refresh")
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

	if minishift {
		openShiftType = "minishift"
	} else if ocp {
		openShiftType = "ocp"
	} else if osio {
		openShiftType = "ocp"
	}

	namespace := util.DefaultNamespaceFromConfig(kubeconfig)

	i := cheinstall.NewInstaller(config, namespace, openShiftType)

	i.OpenShiftType.Install()

	_,cheEndpointURI := util.CheRouteInfo(config, namespace, "che")

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

	log.Infof("Che is available at: %s",cheEndpointURI)
}
