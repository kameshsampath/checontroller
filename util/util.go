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
package util

import (
	"strings"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"fmt"
	"os"
	"os/signal"
	"syscall"

	routeclient "github.com/openshift/origin/pkg/route/generated/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
)

//DefaultNamespaceFromConfig detect the namespace from current kuberenetes context
func DefaultNamespaceFromConfig(kubeconfig *string) string {
	config, err := clientcmd.LoadFromFile(*kubeconfig)
	if err != nil {
		log.Errorf("Unable to get NS from context of config %s \n", err)
	}
	return strings.Split(config.CurrentContext, "/")[0]
}

//IsChePod verifies if the pod is a che pod wit set of Labels
func IsChePod(obj interface{}) bool {

	pod := obj.(*v1.Pod)

	if val, exists := pod.Labels["deploymentconfig"]; exists {
		if val == "che" {
			return true
		}
	}
	return false
}

//CheRouteInfo - returns the Che External URL configured via Route
// returns domain, full route url e.g. example.com http://che-example.com
func CheRouteInfo(config *rest.Config, namespace string, routeName string) (string, string) {
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
	var domain string

	if s := strings.SplitAfterN(host, ".", 2); s != nil && len(s) == 2 {
		domain = s[1]
	}

	var scheme string

	if route.Spec.TLS == nil {
		scheme = "http"
	} else {
		scheme = "https"
	}

	log.Debugf("Domain %s", domain)
	log.Debugf("Route %s", fmt.Sprintf("%s://%s", scheme, host))

	return domain, fmt.Sprintf("%s://%s", scheme, host)

}

//Handles CTRL + C
func HandleSigterm(stopCh chan struct{}) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-signalChan
	log.Infoln("Received signal %s, shutting down", sig)
	close(stopCh)
}
