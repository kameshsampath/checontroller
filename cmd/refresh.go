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
	"os"
	"path/filepath"
	"strings"

	che "github.com/kameshsampath/checontroller/che"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/client-go/util/workqueue"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
}

// refreshCmd represents the refresh command
var (
	flags = pflag.NewFlagSet("", pflag.ExitOnError)

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

	log.Infof("Incluster ? %s", incluster)
	log.Infof("Che  Endpoint URI %s", cheEndpointURI)
	log.Infof("New Stack URI %s", newStackURL)

	var kubeconfig, podNamespace, cheEndpointURI, newStackURL *string

	var incluster *bool

	var clientset *kubernetes.Clientset

	home := homedir.HomeDir()

	log.Debugf("Home Dir :%s\n", home)

	kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")

	//TODO - get the selectors from user
	incluster = flag.Bool("incluster", false, "Whether the application is deployed inside Kubernetes cluster or outside")
	podNamespace = flag.String("namespace", "", "The Kubernetes Namespace to use")
	cheEndpointURI = flag.String("cheEndpointURI", "", "The Che EndpointURI")
	newStackURL = flag.String("newStackURL", "https://raw.githubusercontent.com/redhat-developer/rh-che/master/assembly/fabric8-stacks/src/main/resources/stacks.json", "The New Stacks URL")

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
			*podNamespace = defaultNamespaceFromConfig(kubeconfig)
		}
		//creates clientset
		clientset, err = kubernetes.NewForConfig(config)
		if err != nil {
			log.Fatalf("Unable to build client %s", err)
		}
	}

	log.Infof("Using Namespace: %s", *podNamespace)

	podListWatcher := cache.NewListWatchFromClient(clientset.Core().RESTClient(), "pods",
		*podNamespace, fields.Everything())

	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	indexer, informer := cache.NewIndexerInformer(podListWatcher, &v1.Pod{}, 0, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				if che.IsChePod(obj) {
					log.Infof("Adding Pod %s to queue", key)
					queue.Add(key)
				}
			}
		},
		UpdateFunc: func(obj, newObj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(newObj)
			if err == nil {
				if che.IsChePod(newObj) {
					log.Infof("Updating Pod %s to queue", key)
					queue.Add(key)
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				log.Infof("Deleteing Pod %s from queue", key)
				queue.Add(key)
			}
		},
	}, cache.Indexers{})

	//create controller
	controller := che.NewCheController(indexer, informer, queue, *cheEndpointURI, *newStackURL, *incluster)

	indexer.Add(&v1.Pod{
		ObjectMeta: meta_v1.ObjectMeta{
			Namespace: *podNamespace,
			Labels: map[string]string{
				"deploymentconfig": "che",
			},
		},
	})
	stopCh := make(chan struct{})
	defer close(stopCh)
	go controller.Run(1, stopCh)
	select {}
}

//defaultNamespaceFromConfig detect the namespace from currentContext
func defaultNamespaceFromConfig(kubeconfig *string) string {
	config, err := clientcmd.LoadFromFile(*kubeconfig)
	if err != nil {
		log.Errorf("Unable to get NS from context of config %s \n", err)
	}
	return strings.Split(config.CurrentContext, "/")[0]
}
