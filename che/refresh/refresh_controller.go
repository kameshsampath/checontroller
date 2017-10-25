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

package refresh

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/rest"

	"github.com/kameshsampath/checontroller/util"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//NewController will build a new controller
func NewCheController(cheEndpointURI, podNamespace, newStackURL, appName string, incluster bool, p rest.Interface) *Controller {

	podListWatcher := cache.NewListWatchFromClient(p, "pods", podNamespace, fields.Everything())

	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	indexer, informer := cache.NewIndexerInformer(podListWatcher, &v1.Pod{}, 0, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				if util.IsChePod(appName, obj) {
					log.Debugf("Adding Pod %s to queue", key)
					queue.Add(key)
				}
			}
		},
		UpdateFunc: func(obj, newObj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(newObj)
			if err == nil {
				if util.IsChePod(appName, newObj) {
					log.Debugf("Updating Pod %s to queue", key)
					queue.Add(key)
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				log.Debugf("Deleting Pod %s from queue", key)
				queue.Add(key)
			}
		},
	}, cache.Indexers{})

	//warm up the index
	labels := map[string]string{
		"app":              appName,
		"deploymentconfig": appName,
		"application":      appName,
	}

	log.Debugf("Warming index with labels :%#v", labels)

	indexer.Add(&v1.Pod{
		ObjectMeta: meta_v1.ObjectMeta{
			Namespace: podNamespace,
			Labels:    labels,
		},
	})

	return &Controller{
		indexer:   indexer,
		informer:  informer,
		queue:     queue,
		incluster: incluster,
		refresher: &Config{
			CheEndpointURI: cheEndpointURI,
			NewStackURL:    newStackURL,
		},
	}
}

//Run will run the informer
func (c *Controller) Run(nofThreads int, stopCh chan struct{}) {
	defer runtime.HandleCrash()

	defer c.queue.ShutDown()

	log.Infoln("Starting Che Refresher")

	go c.informer.Run(stopCh)

	if !cache.WaitForCacheSync(stopCh, c.informer.HasSynced) {
		runtime.HandleError(fmt.Errorf("Timedout waiting for caches to sync"))
		return
	}

	for i := 0; i < nofThreads; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh

	log.Infoln("Stopping Che Refresher")
}

func (c *Controller) runWorker() {
	for c.processNextItem() {
	}
}

func (c *Controller) processNextItem() bool {

	key, quit := c.queue.Get()

	if quit {
		return false
	}

	log.Debugf("Processing Key %s", key.(string))

	defer c.queue.Done(key)

	err := c.refreshStacks(key.(string))

	c.handleError(key, err)

	return true
}

func (c *Controller) handleError(key interface{}, err error) {

	if err == nil {
		c.queue.Forget(key)
		return
	}

	//TODO retry ??

	runtime.HandleError(err)

	log.Debugf("Dropping pod %q out of the queue: %v", key, err)
}

func (c *Controller) refreshStacks(key string) error {

	obj, exists, err := c.indexer.GetByKey(key)

	if err != nil {
		log.Errorf("Unable to get object %s from store, %v", key, err)
		return err
	}

	if exists {
		log.Debugf("Pod %s does exist, try to refresh stack", key)
		pod := obj.(*v1.Pod)
		log.Debugf("Pod :%s has state :%s", pod.ObjectMeta.Name, pod.Status.Phase)
		if pod.Status.Phase == "Running" {
			for _, container := range pod.Status.ContainerStatuses {
				log.Debugf("Container Name: %s", container.Name)
				if "che" == container.Name && container.Ready {
					time.Sleep(15 * time.Second) //time for ws agent to warmup
					if c.incluster {
						c.refresher.endpointURI(c.incluster, pod)
						c.refresher.RefreshStacks()
					} else {
						c.refresher.RefreshStacks()
					}
					c.Done = true && c.informer.HasSynced()
				}
			}
		}
	}

	return nil
}

//endpointURI helps in refreshing che stacks when incluster mode using POD IP
func (config *Config) endpointURI(incluster bool, pod *v1.Pod) string {
	log.Infoln("Incluster using POD IP for Che EndPoint")
	appPodIP := pod.Status.PodIP
	config.CheEndpointURI = fmt.Sprintf("http://%s:8080", appPodIP)
	log.Infof("Set Che EndPoint to %s", config.CheEndpointURI)
	return config.CheEndpointURI
}
