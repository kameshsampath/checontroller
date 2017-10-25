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
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"

	tv1 "github.com/openshift/origin/pkg/template/apis/template/v1"
	tplclient "github.com/openshift/origin/pkg/template/generated/clientset"

	ov1 "github.com/openshift/origin/pkg/route/apis/route/v1"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//NewBuilder builds the Template
func NewBuilder() *Objects {
	return &Objects{}
}

//CreateTemplate creates OpenShift template based on TemplateObjects
func (t *Objects) CreateTemplate(config *rest.Config) (*tv1.Template, error) {
	tplclientset, err := tplclient.NewForConfig(config)

	if err != nil {
		log.Errorf("%s", err)
	}

	tc := tplclientset.TemplateV1Client.Templates("openshift")

	template := &tv1.Template{}

	if t.Name == "" {
		t.Name = "che-server-single-user"
	}

	template.ObjectMeta.Name = t.Name
	template.ObjectMeta.Annotations = map[string]string{
		"openshift.io/display-name": "Eclipse Che Server 5.x",
		"description":               "Application template to deploy Eclipse Che server",
		"iconClass":                 "icon-che",
		"tags":                      "IDE,eclipse,che",
		"version":                   "1.0",
	}

	labels := map[string]string{
		"app": "${APPLICATION_NAME}",
	}

	//Parameters for Template
	template.Parameters = t.templateParameters()

	rawExts := make([]runtime.RawExtension, 5+len(t.PVCs), 2*(5+len(t.PVCs)))

	om := metav1.ObjectMeta{
		Name:   "${APPLICATION_NAME}",
		Labels: labels,
	}

	i := 0

	//Make Objects to use of Parameters from Template
	if t.ServiceAccount != nil {
		t.ServiceAccount.APIVersion = "v1"
		t.ServiceAccount.Kind = "ServiceAccount"
		t.ServiceAccount.Annotations = map[string]string{}
		t.ServiceAccount.ObjectMeta = om
		rawExts[i] = runtime.RawExtension{Object: t.ServiceAccount}
	}

	if t.Route != nil {
		i++
		t.Route.APIVersion = "v1"
		t.Route.Kind = "Route"
		t.Route.Annotations = map[string]string{}
		t.Route.ObjectMeta = om
		t.Route.Spec = ov1.RouteSpec{
			To: ov1.RouteTargetReference{
				Kind: "Service",
				Name: "${APPLICATION_NAME}-host",
			}}
		rawExts[i] = runtime.RawExtension{Object: t.Route}
	}

	if t.ConfigMap != nil {
		i++
		t.ConfigMap.APIVersion = "v1"
		t.ConfigMap.Kind = "ConfigMap"
		t.ConfigMap.Annotations = map[string]string{}
		t.ConfigMap.ObjectMeta = om
		t.ConfigMap.Data["hostname-http"] = "${APPLICATION_NAME}.${DOMAIN}"
		t.ConfigMap.Data["openshift-serviceaccountname"] = "${APPLICATION_NAME}"
		t.ConfigMap.Data["maven-mirror-url"] = "${MAVEN_MIRROR_URL}"
		rawExts[i] = runtime.RawExtension{Object: t.ConfigMap}
	}

	if t.Service != nil {
		i++
		t.Service.APIVersion = "v1"
		t.Service.Kind = "Service"
		t.Service.Annotations = map[string]string{}
		t.Service.ObjectMeta = metav1.ObjectMeta{
			Name:   "${APPLICATION_NAME}-host",
			Labels: labels,
		}
		t.Service.Spec.Selector = labels
		rawExts[i] = runtime.RawExtension{Object: t.Service}
	}

	//Worth updating this logic when more volumes are added Che
	//Right now assuming only che-data-workspace, che-data-volume are only
	//volumes used
	for _, pvc := range t.PVCs {
		if pvc != nil {
			i++
			pvc.APIVersion = "v1"
			pvc.Kind = "PersistentVolumeClaim"
			pvc.Annotations = map[string]string{}
			//usually the pvc name will be myapp-che-workspace
			//this will split it in such a way that we get two strings
			//arr[0] = myapp- and arr[1] = che-workspace
			arr := strings.SplitAfterN(pvc.ObjectMeta.Name, "-", 2)
			//use arr[1] with application name
			pvc.ObjectMeta = metav1.ObjectMeta{
				Name:   fmt.Sprintf("${APPLICATION_NAME}-%s", arr[1]),
				Labels: labels,
			}
			rawExts[i] = runtime.RawExtension{Object: pvc}
		}
	}

	if t.DeploymentConfig != nil {
		i++
		t.DeploymentConfig.APIVersion = "v1"
		t.DeploymentConfig.Kind = "DeploymentConfig"
		t.DeploymentConfig.Annotations = map[string]string{}
		t.DeploymentConfig.ObjectMeta = om
		t.DeploymentConfig.Spec.Template.Spec.ServiceAccountName = "${APPLICATION_NAME}"
		t.DeploymentConfig.Spec.Selector = labels
		t.DeploymentConfig.Spec.Template.Labels = labels
		t.DeploymentConfig.Spec.Template.Spec.Volumes[0].Name = "${APPLICATION_NAME}-data-volume"
		t.DeploymentConfig.Spec.Template.Spec.Volumes[0].VolumeSource.PersistentVolumeClaim.ClaimName = "${APPLICATION_NAME}-data-volume"
		//trigger 0 uses ImageChange Trigger
		t.DeploymentConfig.Spec.Triggers[0].ImageChangeParams.From.Name = "che-server:${IMAGE_TAG}"

		for _, c := range t.DeploymentConfig.Spec.Template.Spec.Containers {
			for i, e := range c.Env {
				if e.Name == "APPLICATION_NAME" {
					log.Infof("Setting App Name to ${APPLICATION_NAME}")
					e.Value = "${APPLICATION_NAME}"
				}
				if e.ValueFrom != nil && e.ValueFrom.ConfigMapKeyRef != nil {
					temp := e.ValueFrom.ConfigMapKeyRef.LocalObjectReference
					log.Infof("Setting Configmap Name to ${APPLICATION_NAME}")
					temp.Name = "${APPLICATION_NAME}"
					e.ValueFrom.ConfigMapKeyRef.LocalObjectReference = temp
				}
				c.Env[i] = e
			}
		}

		rawExts[i] = runtime.RawExtension{Object: t.DeploymentConfig}

	}

	template.Objects = rawExts
	tpl, err := tc.Create(template)

	return tpl, err
}

//templateParameters defines the OpenShift template parameters
func (t Objects) templateParameters() []tv1.Parameter {
	var parameters = make([]tv1.Parameter, 7, 20)

	parameters[0] = tv1.Parameter{
		DisplayName: "Application Name",
		Description: "The application name for new che server",
		Name:        "APPLICATION_NAME",
		Value:       "myche",
		Required:    true,
	}

	parameters[1] = tv1.Parameter{
		DisplayName: "Domain Name",
		Description: "The domain name to use with the project, its usually combination of $(minishift ip).nip.io e.g. 192.168.64.10.nip.io",
		Name:        "DOMAIN",
		Value:       t.Domain,
		Required:    true,
	}

	parameters[2] = tv1.Parameter{
		DisplayName: "Che Image Tag",
		Description: "The docker image tag to be used for che, available are latest, nightly",
		Name:        "IMAGE_TAG",
		Value:       t.ImageTag,
		Required:    false,
	}
	parameters[3] = tv1.Parameter{
		DisplayName: "Maven Mirror URL",
		Description: "The Maven Mirror URL to be used for maven builds",
		Name:        "MAVEN_MIRROR_URL",
		Value:       "",
		Required:    false,
	}
	parameters[4] = tv1.Parameter{
		DisplayName: "Enable Che Server Debugging",
		Description: "Che Server Debugging Enabled",
		Name:        "CHE_DEBUGGING_ENABLED",
		Value:       "false",
		Required:    false,
	}
	parameters[5] = tv1.Parameter{
		DisplayName: "GitHub Client Id",
		Description: "The GitHub Client Id that can be used with GitHub oAuth2",
		Name:        "CHE_OAUTH_GITHUB_CLIENTID",
		Value:       "",
		Required:    false,
	}
	parameters[6] = tv1.Parameter{
		DisplayName: "GitHub Client Secret",
		Description: "The GitHub Client Secret that can be used with GitHub oAuth2",
		Name:        "CHE_OAUTH_GITHUB_CLIENTSECRET",
		Value:       "",
		Required:    false,
	}

	return parameters
}
