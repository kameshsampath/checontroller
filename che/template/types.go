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

package template

import (
	oappv1 "github.com/openshift/origin/pkg/apps/apis/apps/v1"
	routev1 "github.com/openshift/origin/pkg/route/apis/route/v1"
	v1 "k8s.io/client-go/pkg/api/v1"
)

//Objects is the Object holder to build templates from che install
type Objects struct {
	Name             string
	Domain           string
	ImageTag         string
	ConfigMap        *v1.ConfigMap
	ServiceAccount   *v1.ServiceAccount
	Service          *v1.Service
	DeploymentConfig *oappv1.DeploymentConfig
	Route            *routev1.Route
	PVCs             []*v1.PersistentVolumeClaim
}
