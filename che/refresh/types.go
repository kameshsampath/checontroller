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
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// Stack represents the id and name of the stack
type Stack struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

//Config of the Refresher which will be used to perform REST operations on Che
type Config struct {
	CheEndpointURI string
	NewStackURL    string
}

// Controller holds informer,  queue, indexer
type Controller struct {
	indexer   cache.Indexer
	informer  cache.Controller
	queue     workqueue.RateLimitingInterface
	incluster bool
	refresher *Config
	Done      bool
}
