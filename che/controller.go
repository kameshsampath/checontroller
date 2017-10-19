/*
Copyright 2017 Kamesh Sampath<kamesh.sampath@hotmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package che

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	log "github.com/sirupsen/logrus"
)

//NewController will build a new controller
func NewCheController(indexer cache.Indexer, informer cache.Controller,
	queue workqueue.RateLimitingInterface, cheEndpointURI string, newStackURL string) *Controller {

	return &Controller{
		indexer:  indexer,
		informer: informer,
		queue:    queue,
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

	log.Infoln("Starting Pod Controller")

	go c.informer.Run(stopCh)

	if !cache.WaitForCacheSync(stopCh, c.informer.HasSynced) {
		runtime.HandleError(fmt.Errorf("Timedout waiting for caches to sync"))
		return
	}

	for i := 0; i < nofThreads; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	log.Infoln("Stopping Pod Controller")
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

	log.Infof("Processing Key %s", key.(string))

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

	log.Infof("Dropping pod %q out of the queue: %v", key, err)
}

func (c *Controller) refreshStacks(key string) error {
	obj, exists, err := c.indexer.GetByKey(key)

	if err != nil {
		log.Errorf("Unable to get object %s from store, %v", key, err)
		return err
	}

	if exists {
		log.Infof("Pod %s does exist, try to refresh stack", key)
		pod := obj.(*v1.Pod)
		log.Infof("Pod :%s has state :%s", pod.ObjectMeta.Name, pod.Status.Phase)
		if pod.Status.Phase == "Running" {
			for _, containers := range pod.Status.ContainerStatuses {
				if "che" == containers.Name && containers.Ready {
					log.Infoln("Starting to refresh stacks ..")
					log.Infof("%s", c.refresher.CheEndpointURI)
					c.refresher.RefreshStacks()
				}
			}
		}
	}
	return nil
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
