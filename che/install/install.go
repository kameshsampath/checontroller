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

package install

import (
	"fmt"
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"

	oappv1 "github.com/openshift/origin/pkg/apps/apis/apps/v1"
	appclient "github.com/openshift/origin/pkg/apps/generated/clientset"

	ov1 "github.com/openshift/origin/pkg/route/apis/route/v1"
	routeclient "github.com/openshift/origin/pkg/route/generated/clientset"

	authv1 "github.com/openshift/origin/pkg/authorization/apis/authorization/v1"
	authclient "github.com/openshift/origin/pkg/authorization/generated/clientset"

	imgv1 "github.com/openshift/origin/pkg/image/apis/image/v1"
	isclient "github.com/openshift/origin/pkg/image/generated/clientset"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	coretypes "k8s.io/client-go/kubernetes/typed/core/v1"

	chetpl "github.com/kameshsampath/checontroller/che/template"
	"github.com/kameshsampath/checontroller/util"

	kapi "k8s.io/kubernetes/pkg/api/v1"
)

const (
	//Kubernetes Core Objects
	objConfigMaps      = "configmaps"
	objServiceAccounts = "serviceaccounts"
	objService         = "services"
	objPVC             = "persistentvolumeclaims"

	//OpenShift custom objects
	objDeploymentConfigs = "deploymentconfigs.apps.openshift.io"
	objRoute             = "routes.route.openshift.io"
	objRoleBindings      = "rolebinding.authorization.openshift.io"
	objImageStreams      = "imagestreams.image.openshift.io"
	objTemplates         = "templates.template.openshift.io"

	//
	adminUserRegx = `^.*User "(?P<user>[a-zA-Z0-9_-]*)" cannot create.*$`
)

var (
	//Installer represents the configuration that is provided during Che Installation
	Installer          *InstallerConfig
	validAdminUserReqd = regexp.MustCompile(adminUserRegx)
	tplBuilder         = chetpl.NewBuilder()
)

//Install - starts the installation process of Che
func (ot OpenShiftType) Install() {

	tplBuilder.ImageTag = Installer.ImageTag

	pvcs := make([]*v1.PersistentVolumeClaim, 2)

	tplBuilder.ServiceAccount = Installer.createCheServiceAccount()
	tplBuilder.Route = Installer.createCheRoute()

	Installer.createImageStream()

	//determine the domain
	domain, _ := util.CheRouteInfo(Installer.Config, Installer.Namespace, Installer.AppName)
	Installer.Domain = domain
	tplBuilder.Domain = domain

	tplBuilder.ConfigMap = Installer.createCheConfigMap()

	pvcs[0] = Installer.createCheDataPVC()
	pvcs[1] = Installer.createCheWorkspacePVC()
	tplBuilder.PVCs = pvcs

	tplBuilder.Service = Installer.createCheService()

	switch ot {
	case "minishift":
		//Installer.applyQuota()
		Installer.createRoleBinding()
	case "ocp":
		//Installer.applyQuota()
	}

	tplBuilder.DeploymentConfig = Installer.createCheDeploymentConfig()

	if Installer.SaveAsTemplate != "" {
		log.Infof(`Saving Che Install as Template with name "%s"`, Installer.SaveAsTemplate)
		tplBuilder.Name = Installer.SaveAsTemplate
		_, err := tplBuilder.CreateTemplate(Installer.Config)
		if err != nil {
			b := objectExists(objTemplates, Installer.SaveAsTemplate, err)
			if b {
				log.Infoln(`Template  "%s" already exists, skipping creation`, Installer.SaveAsTemplate)
				return
			} else if match := validAdminUserReqd.FindStringSubmatch(err.Error()); match != nil {
				log.Fatalf(`Logged in user "%s" does not have enough privileges to create template in openshift namespace, login as Admin user or similar`, match[1])
				return
			} else {
				log.Errorf(`Error saving as template "%s" %s`, Installer.SaveAsTemplate, err)
			}
		} else {
			log.Infof(`Successfully created template "%s"`, Installer.SaveAsTemplate)
		}
	}
}

//Creates Eclipse Che Server Image Stream
func (i *InstallerConfig) createImageStream() {
	clientset, err := isclient.NewForConfig(i.Config)

	if err != nil {
		log.Fatalf("%s", err)
	}

	imgc := clientset.ImageStreams("openshift")

	_, err = imgc.Create(&imgv1.ImageStream{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "che-server",
			Annotations: map[string]string{"description": "Eclipse Che Server 5.x"},
		},
		Spec: imgv1.ImageStreamSpec{
			Tags: []imgv1.TagReference{
				{
					Name: "latest",
					Annotations: map[string]string{
						"name":        " Eclipse Che Server (latest)",
						"description": "Eclipse Che server centos images.",
						"iconClass":   "icon-che",
						"tags":        "java,ide,eclipse,che",
					},
					From: &kapi.ObjectReference{
						Name: "latest-centos",
						Kind: "ImageStreamTag",
					},
				},
				{
					Name: "latest-centos",
					Annotations: map[string]string{
						"openshift.io/display-name": "Eclipse Che Server Stable Latest",
						"description":               "Eclipse Che server centos images.",
						"iconClass":                 "icon-che",
						"tags":                      "java,ide,eclipse,che",
						"version":                   "5.x-centos",
					},
					From: &kapi.ObjectReference{
						Name: "eclipse/che-server:latest-centos",
						Kind: "DockerImage",
					},
				},
				{
					Name: "nightly",
					Annotations: map[string]string{
						"name":        " Eclipse Che Server (nightly)",
						"description": "Eclipse Che server nightly centos images.",
						"iconClass":   "icon-che",
						"tags":        "java,ide,eclipse,che",
					},
					From: &kapi.ObjectReference{
						Name: "nightly-centos",
						Kind: "ImageStreamTag",
					},
				},
				{
					Name: "nightly-centos",
					Annotations: map[string]string{
						"openshift.io/display-name": "Eclipse Che Server 5.x.x-SNAPSHOT",
						"description":               "Eclipse Che server nightly centos images.",
						"iconClass":                 "icon-che",
						"tags":                      "java,ide,eclipse,che",
						"version":                   "nightly-centos",
					},
					From: &kapi.ObjectReference{
						Name: "eclipse/che-server:nightly-centos",
						Kind: "DockerImage",
					},
				},
			},
		},
	})

	if err != nil {
		b := objectExists(objImageStreams, "che-server", err)
		if b {
			log.Infoln(`ImageStream  "che-server" already exists, skipping creation`)
			return
		} else if match := validAdminUserReqd.FindStringSubmatch(err.Error()); match != nil {
			log.Fatalf(`Logged in user "%s" does not have enough priviliges to create ImageStream on Namespace "openshift",
				 login as Admin user or similar`, match[1])
			return
		} else {
			log.Fatalf(`Error creating ImageStream "che" %s`, err)
		}
	} else {
		log.Infoln(`ImageStream "che-server" successfully created`)
	}
}

//createRoleBinding creates the RoleBinding required for Che ServiceAccount
func (i *InstallerConfig) createRoleBinding() {
	clientset, err := authclient.NewForConfig(i.Config)

	if err != nil {
		log.Fatalf("%s", err)
	}

	ac := clientset.RoleBindings(i.Namespace)

	rb := fmt.Sprintf(`%s-che`, i.AppName)

	_, err = ac.Create(&authv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: rb,
			Labels: map[string]string{
				"app": i.AppName,
			},
		},
		RoleRef: kapi.ObjectReference{
			Name: "admin",
		},
		Subjects: []kapi.ObjectReference{
			{
				Name: i.AppName,
				Kind: "ServiceAccount",
			},
		},
	})

	if err != nil {
		b := objectExists(objRoleBindings, rb, err)
		if b {
			log.Infof(`RoleBinding  "%s" already exists, skipping creation`, rb)
			return
		} else if match := validAdminUserReqd.FindStringSubmatch(err.Error()); match != nil {
			log.Fatalf(`Logged in user "%s" does not have enough privileges to create RoleBinding, login as Admin user or similar`, match[1])
			return
		} else {
			log.Fatalf(`Error creating RoleBinding "%s" %s`, rb, err)
		}
	} else {
		log.Infof(`RoleBinding "%s" successfully created`, rb)
	}
}

//createConfigMap Creates the che configMap
func (i *InstallerConfig) createCheConfigMap() *v1.ConfigMap {
	clientset, err := kubernetes.NewForConfig(i.Config)

	if err != nil {
		log.Fatalf("%s", err)
	}

	cmClient := clientset.ConfigMaps(i.Namespace)

	val := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: i.AppName,
			Labels: map[string]string{
				"app": i.AppName,
			},
		},
		Data: i.OpenShiftType.configMap(),
	}

	_, err = cmClient.Create(val)

	if err != nil {
		b := objectExists("configmaps", i.AppName, err)
		if b {
			log.Infof(`ConfigMap  "%s" already exists, skipping creation`, i.AppName)
			return val
		}
		log.Fatalf(`Error creating ConfigMap "%s" %s`, i.AppName, err)
	} else {
		log.Infof(`ConfigMap "%s" successfully created`, i.AppName)
	}

	return val
}

//createCheDataPVC Creates the che data Persistence Volume Claim
func (i *InstallerConfig) createCheDataPVC() *v1.PersistentVolumeClaim {
	clientset, err := kubernetes.NewForConfig(i.Config)

	if err != nil {
		log.Fatalf("%s", err)
	}

	pvcClient := clientset.CoreV1().PersistentVolumeClaims(i.Namespace)

	val, err := createDefaultPVC(pvcClient, i.AppName, fmt.Sprintf("%s-data-volume", i.AppName))

	if err != nil {
		b := objectExists(objPVC, fmt.Sprintf("%s-data-volume", i.AppName), err)
		if b {
			log.Infof(`PersistenceVolumeClaim  "%s" already exists, skipping creation`, fmt.Sprintf("%s-data-volume", i.AppName))
			return val
		}
		log.Fatalf(`Error creating PersistenceVolumeClaim "%s" %s`, fmt.Sprintf("%s-data-volume", i.AppName), err)
	} else {
		log.Infof(`PersistenceVolumeClaim "%s" successfully created`, fmt.Sprintf("%s-data-volume", i.AppName))
	}
	return val
}

//createCheWorkspacePVC Creates the che workspace Persistence Volume Claim
func (i *InstallerConfig) createCheWorkspacePVC() *v1.PersistentVolumeClaim {
	clientset, err := kubernetes.NewForConfig(i.Config)

	if err != nil {
		log.Fatalf("%s", err)
	}

	pvcClient := clientset.CoreV1().PersistentVolumeClaims(i.Namespace)

	val, err := createDefaultPVC(pvcClient, i.AppName, fmt.Sprintf("%s-che-workspace", i.AppName))

	if err != nil {
		b := objectExists(objPVC, fmt.Sprintf("%s-che-workspace", i.AppName), err)
		if b {
			log.Infof(`PersistenceVolumeClaim  "%s" already exists, skipping creation`, fmt.Sprintf("%s-che-workspace", i.AppName))
			return val
		}
		log.Fatalf(`Error creating PersistenceVolumeClaim "%s" %s`, fmt.Sprintf("%s-che-workspace", i.AppName), err)
	} else {
		log.Infof(`PersistenceVolumeClaim "%s" successfully created`, fmt.Sprintf("%s-che-workspace", i.AppName))
	}

	return val
}

//createDeploymentConfig Creates the che deploymentconfig
func (i *InstallerConfig) createCheDeploymentConfig() *oappv1.DeploymentConfig {
	appcs, err := appclient.NewForConfig(i.Config)

	if err != nil {
		log.Fatalf("%s", err)
	}

	dcClient := appcs.AppsV1().DeploymentConfigs(i.Namespace)

	ts := int64(10000)

	val := &oappv1.DeploymentConfig{ObjectMeta: metav1.ObjectMeta{
		Name: i.AppName,
		Labels: map[string]string{
			"app": i.AppName,
		},
	}, Spec: oappv1.DeploymentConfigSpec{
		Replicas: 1,
		Selector: map[string]string{
			"app": i.AppName,
		},
		Strategy: oappv1.DeploymentStrategy{
			RecreateParams: &oappv1.RecreateDeploymentStrategyParams{
				TimeoutSeconds: &ts,
			},
			Type: "Recreate",
		},
		Template: &kapi.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"app": i.AppName,
			},
		}, Spec: kapi.PodSpec{
			Containers: []kapi.Container{
				{
					Name:            "che",
					Env:             i.OpenShiftType.cheEnvVars(),
					Image:           "che-server",
					ImagePullPolicy: kapi.PullIfNotPresent,
					Ports: []kapi.ContainerPort{
						{
							ContainerPort: 8080,
							Name:          "http",
						},
						{
							ContainerPort: 8080,
							Name:          "http-debug",
						},
					},
					LivenessProbe: &kapi.Probe{
						InitialDelaySeconds: 120,
						TimeoutSeconds:      10,
						Handler: kapi.Handler{
							HTTPGet: &kapi.HTTPGetAction{
								Path:   "api/system/state",
								Port:   intstr.IntOrString{IntVal: 8080},
								Scheme: kapi.URISchemeHTTP,
							},
						},
					},
					ReadinessProbe: &kapi.Probe{
						InitialDelaySeconds: 15,
						TimeoutSeconds:      60,
						Handler: kapi.Handler{
							HTTPGet: &kapi.HTTPGetAction{
								Path:   "api/system/state",
								Port:   intstr.IntOrString{IntVal: 8080},
								Scheme: kapi.URISchemeHTTP,
							},
						},
					},
					Resources: kapi.ResourceRequirements{
						Limits: kapi.ResourceList{
							"memory": resource.MustParse("600Mi"),
						},
						Requests: kapi.ResourceList{
							"memory": resource.MustParse("256Mi"),
						},
					},
				},
			},
			ServiceAccountName: i.AppName,
			Volumes: []kapi.Volume{
				{
					Name: fmt.Sprintf("%s-data-volume", i.AppName),
					VolumeSource: kapi.VolumeSource{
						PersistentVolumeClaim: &kapi.PersistentVolumeClaimVolumeSource{
							ClaimName: fmt.Sprintf("%s-data-volume", i.AppName),
						},
					},
				},
			},
		}},
		Triggers: oappv1.DeploymentTriggerPolicies{
			oappv1.DeploymentTriggerPolicy{
				Type: oappv1.DeploymentTriggerOnImageChange,
				ImageChangeParams: &oappv1.DeploymentTriggerImageChangeParams{
					Automatic:      true,
					ContainerNames: []string{"che"},
					From: kapi.ObjectReference{
						Kind:      "ImageStreamTag",
						Namespace: "openshift",
						Name:      fmt.Sprintf(`che-server:%s`, i.ImageTag),
					},
				},
			},
			oappv1.DeploymentTriggerPolicy{
				Type: oappv1.DeploymentTriggerOnConfigChange,
			},
		},
	}}

	_, err = dcClient.Create(val)

	if err != nil {
		b := objectExists(objDeploymentConfigs, i.AppName, err)
		if b {
			log.Infof(`DeploymentConfig  "%s" already exists, skipping creation`, i.AppName)
			return val
		}
		log.Fatalf(`Error creating DeploymentConfig "%s" %s`, i.AppName, err)
	} else {
		log.Infof(`DeploymentConfig "%s" successfully created`, i.AppName)
	}

	return val
}

//createCheRoute Creates the che route
func (i *InstallerConfig) createCheRoute() *ov1.Route {

	rc, err := routeclient.NewForConfig(i.Config)

	if err != nil {
		log.Fatalf("%s", err)
	}

	routesClient := rc.RouteV1Client.Routes(i.Namespace)

	val := &ov1.Route{ObjectMeta: metav1.ObjectMeta{
		Name: i.AppName,
		Labels: map[string]string{
			"app": i.AppName,
		},
	}, Spec: i.OpenShiftType.routeSpec()}

	_, err = routesClient.Create(val)

	if err != nil {
		b := objectExists(objRoute, i.AppName, err)
		if b {
			log.Infof(`Route "%s" already exists, skipping creation`, i.AppName)
			return val
		}
		log.Fatalf(`Error creating Route "%s" %s`, i.AppName, err)
	} else {
		log.Infof(`Route "%s" successfully created`, i.AppName)
	}

	return val
}

//createCheService Creates the che service
func (i *InstallerConfig) createCheService() *v1.Service {
	clientset, err := kubernetes.NewForConfig(i.Config)

	if err != nil {
		log.Fatalf("%s", err)
	}

	svcClient := clientset.CoreV1().Services(i.Namespace)

	val := &v1.Service{ObjectMeta: metav1.ObjectMeta{
		Name: fmt.Sprintf(`%s-host`, i.AppName),
		Labels: map[string]string{
			"app": i.AppName,
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
			"app": i.AppName,
		},
	}}
	_, err = svcClient.Create(val)

	if err != nil {
		b := objectExists(objService, fmt.Sprintf(`%s-host`, i.AppName), err)
		if b {
			log.Infof(`Service "%s" already exists, skipping creation`, fmt.Sprintf(`%s-host`, i.AppName))
			return val
		}
		log.Fatalf(`Error creating Service "%s" %s`, fmt.Sprintf(`%s-host`, i.AppName), err)
	} else {
		log.Infof(`Service "%s" successfully created`, fmt.Sprintf(`%s-host`, i.AppName))
	}

	return val
}

//createCheServiceAccount Creates the che service account
func (i *InstallerConfig) createCheServiceAccount() *v1.ServiceAccount {

	clientset, err := kubernetes.NewForConfig(i.Config)

	if err != nil {
		log.Fatalf("%s", err)
	}

	saClient := clientset.CoreV1().ServiceAccounts(i.Namespace)

	val := &v1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{
		Name: i.AppName,
		Labels: map[string]string{
			"app": i.AppName,
		},
	}}
	_, err = saClient.Create(val)

	if err != nil {
		b := objectExists(objServiceAccounts, i.AppName, err)
		if b {
			log.Infof(`ServiceAccount "%s" already exists, skipping creation`, i.AppName)
			return val
		}
		log.Fatalf(`Error creating ServiceAccount "%s" %s`, i.AppName, err)
	} else {
		log.Infof(`ServiceAccount "%s" successfully created`, i.AppName)
	}

	return val
}

//createDefaultPVC creates a defaultPVC
func createDefaultPVC(pvcClient coretypes.PersistentVolumeClaimInterface, appName, volumeName string) (*v1.PersistentVolumeClaim, error) {
	val := &v1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{
		Name: volumeName,
		Labels: map[string]string{
			"app": appName,
		},
	}, Spec: v1.PersistentVolumeClaimSpec{
		AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
		Resources: v1.ResourceRequirements{
			Requests: map[v1.ResourceName]resource.Quantity{
				"storage": resource.MustParse("1Gi"),
			},
		},
	}}

	_, err := pvcClient.Create(val)

	return val, err
}

//objectExists simple method to check if the object exists or not
func objectExists(object, name string, err error) bool {
	return strings.Contains(err.Error(), fmt.Sprintf(`%s "%s" already exists`, object, name))
}

// build the Che configmap based on the OpenShiftType
func (ot OpenShiftType) configMap() map[string]string {
	cm := map[string]string{
		"hostname-http":                                                  fmt.Sprintf(`%s.%s`, Installer.AppName, Installer.Domain),
		"workspace-storage":                                              "/home/user/che/workspaces",
		"workspace-storage-create-folders":                               "false",
		"local-conf-dir":                                                 "/etc/conf",
		"openshift-serviceaccountname":                                   Installer.AppName,
		"che-server-evaluation-strategy":                                 "docker-local-custom",
		"che.logs.dir":                                                   "/data/logs",
		"che.docker.server_evaluation_strategy.custom.template":          "<serverName>-<if(isDevMachine)><workspaceIdWithoutPrefix><else><machineName><endif>-<externalAddress>",
		"che.docker.server_evaluation_strategy.custom.external.protocol": "https",
		"che.predefined.stacks.reload_on_start":                          "true",
		"log-level":                                                      "INFO",
		"docker-connector":                                               "openshift",
		"port":                                                           "8080",
		"remote-debugging-enabled":         "false",
		"che-oauth-github-forceactivation": "true",
		"workspaces-memory-limit":          "1900Mi",
		"workspaces-memory-request":        "1100Mi",
		"enable-workspaces-autostart":      "false",
		"che-server-java-opts":             "-XX:+UseG1GC -XX:+UseStringDeduplication -XX:MinHeapFreeRatio=20 -XX:MaxHeapFreeRatio=40 -XX:MaxRAM=600m -Xms256m",
		"che-workspaces-java-opts":         "-XX:+UseG1GC -XX:+UseStringDeduplication -XX:MinHeapFreeRatio=20 -XX:MaxHeapFreeRatio=40 -XX:MaxRAM=1200m -Xms256m",
		"che-openshift-secure-routes":      "true",
		"che-secure-external-urls":         "true",
		"che-server-timeout-ms":            "3600000",
		"che-openshift-precreate-subpaths": "false",
		"che-workspace-auto-snapshot":      "false",
		"keycloak-disabled":                "false",
		"maven-mirror-url":                 Installer.MavenMirrorURL,
	}

	switch ot {
	case "minishift":
		cm["che.docker.server_evaluation_strategy.custom.external.protocol"] = "http"
		cm["che.predefined.stacks.reload_on_start"] = "false"
		cm["che-openshift-secure-routes"] = "false"
		cm["che-secure-external-urls"] = "false"
		cm["che-openshift-precreate-subpath"] = "true"
		cm["keycloak-disabled"] = "true"
		cm["workspaces-memory-limit"] = "1300Mi"
		cm["workspaces-memory-request"] = "500Mi"
	case "ocp":
		cm["keycloak-oso-endpoint"] = "${KEYCLOAK_OSO_ENDPOINT}"
		cm["keycloak-github-endpoint"] = "${KEYCLOAK_GITHUB_ENDPOINT}"
		cm["keycloak-disabled"] = "false"
	}

	log.Debugf("ConfigMap Data: %s", cm)
	return cm
}

//routeSpec builds route spec based on type of OpenShift cluster
func (ot OpenShiftType) routeSpec() ov1.RouteSpec {
	routeSpec := &ov1.RouteSpec{
		To: ov1.RouteTargetReference{
			Kind: "Service",
			Name: fmt.Sprintf(`%s-host`, Installer.AppName),
		}}

	switch ot {
	case "minishift":
		log.Infoln("No Extra Route Config for minishift and default cases")
	case "ocp":
		routeSpec.TLS = &ov1.TLSConfig{
			InsecureEdgeTerminationPolicy: "Redirect",
			Termination:                   "edge",
		}
	default:
		log.Infoln("No Extra Route Config for minishift and default cases")
	}

	log.Debugf("Route Spec : %#v", *routeSpec)

	return *routeSpec
}

//CheEnvVars Customized Environment Variables for each OpenShiftType
func (ot OpenShiftType) cheEnvVars() []kapi.EnvVar {
	switch ot {
	case "minishift":
		return Installer.CheEnvVars()
	case "ocp":
		return Installer.OCPCheEnvVars()
	default:
		return Installer.CheEnvVars()
	}
}
