package cmd

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	util "github.com/kameshsampath/checontroller/util"
	routeclient "github.com/openshift/origin/pkg/route/generated/clientset"

	ov1 "github.com/openshift/origin/pkg/route/apis/route/v1"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	coretypes "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/pkg/api/v1"
	rest "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// openshiftType to distinguish between minishift, ocp, osio etc.,
type OpenshiftType string

//installerConfig
type installerConfig struct {
	config        *rest.Config
	namespace     string
	openshiftType OpenshiftType
}

const (
	objConfigMaps        = "configmaps"
	objServiceAccounts   = "serviceaccounts"
	objService           = "services"
	objDeploymentConfigs = "deploymentconfigs"
	objPVC               = "persistentvolumeclaims"
	objRoute             = "routes.route.openshift.io"
)

// installCmd represents the refresh command
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

	installCmd.Flags().BoolVarP(&refreshStacks, "refreshstacks", "r", false, "Refresh the stack to make it OpenShift compatible")
	installCmd.Flags().BoolVarP(&minishift, "minishift", "m", false, "Is OpenShift cluster running on minishift")
	installCmd.Flags().BoolVarP(&osio, "osio", "o", false, "Is OpenShift cluster running on openshift.io")
	installCmd.Flags().BoolVarP(&ocp, "ocp", "d", false, "Is OpenShift cluster running on OpenShift Container Platform")
	installCmd.Flags().StringVarP(&newStacksURL, "newStackURL", "n", "https://raw.githubusercontent.com/redhat-developer/rh-che/master/assembly/fabric8-stacks/src/main/resources/stacks.json",
		"The JSON from where to load the new stacks during refresh")
}

func install(cmd *cobra.Command, args []string) {
	log.Infoln("Starting Che Install on OpenShift")

	var kubeconfig *string

	home := homedir.HomeDir()

	log.Infof("Home Dir :%s\n", home)

	kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)

	if err != nil {
		log.Fatalf("Unable to build config, %v", err)
	}

	if err != nil {
		log.Fatalf("%s", err)
	}

	var openShiftType OpenshiftType

	if minishift {
		openShiftType = "minishift"
	} else if ocp {
		openShiftType = "ocp"
	} else if osio {
		openShiftType = "ocp"
	}

	i := &installerConfig{
		config:        config,
		namespace:     util.DefaultNamespaceFromConfig(kubeconfig),
		openshiftType: openShiftType,
	}

	i.createCheServiceAccount()
	i.createCheConfigMap()
	i.createCheDataPVC()
	i.createCheWorkspacePVC()
	i.createCheService()
	i.createCheRoute()

}

//createConfigMap Creates the che configMap
func (i *installerConfig) createCheConfigMap() {
	clientset, err := kubernetes.NewForConfig(i.config)

	if err != nil {
		log.Fatalf("%s", err)
	}

	cmClient := clientset.ConfigMaps(i.namespace)

	_, err = cmClient.Create(&v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "che",
			Labels: map[string]string{
				"app": "che",
			},
		},
		Data: map[string]string{
			"chant": "Hare Krishna!",
		},
	})

	if err != nil {
		b := objectExists("configmaps", "che", err)
		if b {
			log.Infoln("ConfigMap  \"che\" already exists, skipping creation")
			return
		}
		log.Fatalf("Error creating ConfigMap \"che\" %s", err)
	}
}

//createCheDataPVC Creates the che data Persistence Volume Claim
func (i *installerConfig) createCheDataPVC() {
	clientset, err := kubernetes.NewForConfig(i.config)

	if err != nil {
		log.Fatalf("%s", err)
	}

	pvcClient := clientset.CoreV1().PersistentVolumeClaims(i.namespace)

	_, err = createDefaultPVC(pvcClient, "che-data-volume")

	if err != nil {
		b := objectExists(objPVC, "che-data-volume", err)
		if b {
			log.Infoln("PersistenceVolumeClaim  \"che-data-volume\" already exists, skipping creation")
			return
		}
		log.Fatalf("Error creating PersistenceVolumeClaim \"che-data-volume\" %s", err)
	}
}

//createCheWorkspacePVC Creates the che workspace Persistence Volume Claim
func (i *installerConfig) createCheWorkspacePVC() {
	clientset, err := kubernetes.NewForConfig(i.config)

	if err != nil {
		log.Fatalf("%s", err)
	}

	pvcClient := clientset.CoreV1().PersistentVolumeClaims(i.namespace)

	_, err = createDefaultPVC(pvcClient, "claim-che-workspace")

	if err != nil {
		b := objectExists(objPVC, "claim-che-workspace", err)
		if b {
			log.Infoln("PersistenceVolumeClaim  \"claim-che-workspace\" already exists, skipping creation")
			return
		}
		log.Fatalf("Error creating PersistenceVolumeClaim \"claim-che-workspace\" %s", err)
	}
}

//createDeploymentConfig Creates the che deploymentconfig
func createCheDeploymentConfig(config *rest.Config) {
	return
}

//createCheRoute Creates the che route
func (i *installerConfig) createCheRoute() {
	rc, err := routeclient.NewForConfig(i.config)

	if err != nil {
		log.Fatalf("%s", err)
	}

	routesClient := rc.RouteV1Client.Routes(i.namespace)

	_, err = routesClient.Create(&ov1.Route{ObjectMeta: metav1.ObjectMeta{
		Name: "che",
		Labels: map[string]string{
			"app": "che",
		},
	}, Spec: i.openshiftType.routeSpec()})

	if err != nil {
		b := objectExists(objRoute, "che", err)
		if b {
			log.Infoln("Route \"che\" already exists, skipping creation")
			return
		}
		log.Fatalf("Error creating Route \"che\" %s", err)
	}
}

//createCheService Creates the che service
func (i *installerConfig) createCheService() {
	clientset, err := kubernetes.NewForConfig(i.config)

	if err != nil {
		log.Fatalf("%s", err)
	}

	svcClient := clientset.CoreV1().Services(i.namespace)

	_, err = svcClient.Create(&v1.Service{ObjectMeta: metav1.ObjectMeta{
		Name: "che-host",
		Labels: map[string]string{
			"app": "che",
		},
	}, Spec: v1.ServiceSpec{Ports: []v1.ServicePort{
		v1.ServicePort{
			Name:     "http",
			Port:     8080,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				IntVal: 8080,
			}},
	},
		Selector: map[string]string{
			"app": "che",
		},
	}})

	if err != nil {
		b := objectExists(objService, "che-host", err)
		if b {
			log.Infoln("Service \"che-host\" already exists, skipping creation")
			return
		}
		log.Fatalf("Error creating Service \"che\" %s", err)
	}
}

//createCheServiceAccount Creates the che service account
func (i *installerConfig) createCheServiceAccount() {

	clientset, err := kubernetes.NewForConfig(i.config)

	if err != nil {
		log.Fatalf("%s", err)
	}

	saClient := clientset.CoreV1().ServiceAccounts(i.namespace)

	_, err = saClient.Create(&v1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{
		Name: "che",
		Labels: map[string]string{
			"app": "che",
		},
	}})

	if err != nil {
		b := objectExists(objServiceAccounts, "che", err)
		if b {
			log.Infoln("ServiceAccount \"che\" already exists, skipping creation")
			return
		}
		log.Fatalf("Error creating ServiceAccount \"che\" %s", err)
	}
}

//objectExists simple method to check if the object exists or not
func objectExists(object, name string, err error) bool {
	return strings.Contains(err.Error(), fmt.Sprintf("%s \"%s\" already exists", object, name))
}

//createDefaultPVC creates a defaultPVC
func createDefaultPVC(pvcClient coretypes.PersistentVolumeClaimInterface, volumeName string) (*v1.PersistentVolumeClaim, error) {
	return pvcClient.Create(&v1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{
		Name: volumeName,
		Labels: map[string]string{
			"app": "che",
		},
	}, Spec: v1.PersistentVolumeClaimSpec{
		AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
		Resources: v1.ResourceRequirements{
			Requests: map[v1.ResourceName]resource.Quantity{
				"storage": resource.MustParse("1Gi"),
			},
		},
	}})
}

//routeSpec builds route spec based on type of OpenShift cluster
func (ot OpenshiftType) routeSpec() ov1.RouteSpec {
	routeSpec := &ov1.RouteSpec{
		To: ov1.RouteTargetReference{
			Kind: "Service",
			Name: "che-host",
		},
	}
	switch ot {
	case "osio":
		routeSpec.TLS = &ov1.TLSConfig{
			InsecureEdgeTerminationPolicy: "Redirect",
			Termination:                   "edge",
		}
	case "ocp":
		routeSpec.TLS = &ov1.TLSConfig{
			InsecureEdgeTerminationPolicy: "Redirect",
			Termination:                   "edge",
		}
	default:
		log.Infoln("No Extra config for minishift and default cases")
	}

	log.Debugf("Route Spec : %#v", *routeSpec)

	return *routeSpec
}
