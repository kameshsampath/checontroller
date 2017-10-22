package util

import (
	"strings"

	log "github.com/sirupsen/logrus"
	rest "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	routeclient "github.com/openshift/origin/pkg/route/generated/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//DefaultNamespaceFromConfig detect the namespace from current kuberenetes context
func DefaultNamespaceFromConfig(kubeconfig *string) string {
	config, err := clientcmd.LoadFromFile(*kubeconfig)
	if err != nil {
		log.Errorf("Unable to get NS from context of config %s \n", err)
	}
	return strings.Split(config.CurrentContext, "/")[0]
}

//CheExternalURLFromRoute - retuns the Che External URL configured via Route
func CheExternalURLFromRoute(config *rest.Config, namespace string, routeName string) string {
	if routeName == "" {
		routeName = "che"
	}
	rc, err := routeclient.NewForConfig(config)

	if err != nil {
		log.Fatalf("%s", err)
	}

	routesClient := rc.RouteV1Client.Routes(namespace)
	route, err := routesClient.Get(routeName, metav1.GetOptions{})

	if err != nil {
		log.Errorf("Unable to get the route %s", err)
	}

	host := route.Spec.Host
	log.Infof("Che Route URL :%s", host)

	return host
}
