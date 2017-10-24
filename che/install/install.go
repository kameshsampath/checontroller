package install

import (
	"fmt"
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"

	"k8s.io/client-go/rest"

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

	//
	adminUserRegx = `^.*User "(?P<user>[a-zA-Z0-9_-]*)" cannot create.*$`
)

var (
	installer          *InstallerConfig
	validAdminUserReqd = regexp.MustCompile(adminUserRegx)
)

//NewInstaller contructs a new installer an
func NewInstaller(config *rest.Config, namespace string, openshiftType OpenShiftType) *InstallerConfig {
	installer = &InstallerConfig{
		config:        config,
		namespace:     namespace,
		OpenShiftType: openshiftType,
	}

	return installer
}

//Install - starts the installation process of Che
func (ot OpenShiftType) Install() {
	installer.createCheServiceAccount()
	installer.createCheRoute()

	//determine the domain
	domain, _ := util.CheRouteInfo(installer.config, installer.namespace, "che")

	installer.createCheConfigMap(installer.namespace, domain)
	installer.createCheDataPVC()
	installer.createCheWorkspacePVC()
	installer.createCheService()

	switch ot {
	case "osio":
		//installer.applyQuota()
	case "ocp":
		installer.createImageStream()
	default: // minishift
		//installer.applyQuota()
		installer.createImageStream()
		installer.createRoleBinding()
	}
	installer.createCheDeploymentConfig()
}

//Creates Eclipse Che Server Image Stream
func (i *InstallerConfig) createImageStream() {
	clientset, err := isclient.NewForConfig(i.config)

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
						Name: "5.18.0-centos",
						Kind: "ImageStreamTag",
					},
				},
				{
					Name: "5.18.0-centos",
					Annotations: map[string]string{
						"openshift.io/display-name": "Eclipse Che Server  5.18.0",
						"description":               "Eclipse Che server centos images.",
						"iconClass":                 "icon-che",
						"tags":                      "java,ide,eclipse,che",
						"version":                   "5.18.0-centos",
					},
					From: &kapi.ObjectReference{
						Name: "eclipse/che-server:5.18.0-centos",
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
						"openshift.io/display-name": "Eclipse Che Server 5.19.0-SNAPSHOT",
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
	clientset, err := authclient.NewForConfig(i.config)

	if err != nil {
		log.Fatalf("%s", err)
	}

	ac := clientset.RoleBindings(i.namespace)

	_, err = ac.Create(&authv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "che",
			Labels: map[string]string{
				"app": "che",
			},
		},
		RoleRef: kapi.ObjectReference{
			Name: "admin",
		},
		Subjects: []kapi.ObjectReference{
			{
				Name: "che",
				Kind: "ServiceAccount",
			},
		},
	})

	if err != nil {
		b := objectExists(objRoleBindings, "che", err)
		if b {
			log.Infoln(`RoleBinding  "che" already exists, skipping creation`)
			return
		} else if match := validAdminUserReqd.FindStringSubmatch(err.Error()); match != nil {
			log.Fatalf(`Logged in user "%s" does not have enough priviliges, login as Admin user or similar`, match[1])
			return
		} else {
			log.Fatalf(`Error creating RoleBinding "che" %s`, err)
		}
	} else {
		log.Infoln(`RoleBinding "che" successfully created`)
	}
}

//createConfigMap Creates the che configMap
func (i *InstallerConfig) createCheConfigMap(project, domain string) {
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
		Data: i.OpenShiftType.configMap(project, domain),
	})

	if err != nil {
		b := objectExists("configmaps", "che", err)
		if b {
			log.Infoln(`ConfigMap  "che" already exists, skipping creation`)
			return
		}
		log.Fatalf("Error creating ConfigMap \"che\" %s", err)
	} else {
		log.Infoln(`ConfigMap "che" successfully created`)
	}
}

//createCheDataPVC Creates the che data Persistence Volume Claim
func (i *InstallerConfig) createCheDataPVC() {
	clientset, err := kubernetes.NewForConfig(i.config)

	if err != nil {
		log.Fatalf("%s", err)
	}

	pvcClient := clientset.CoreV1().PersistentVolumeClaims(i.namespace)

	_, err = createDefaultPVC(pvcClient, "che-data-volume")

	if err != nil {
		b := objectExists(objPVC, "che-data-volume", err)
		if b {
			log.Infoln(`PersistenceVolumeClaim  "che-data-volume" already exists, skipping creation`)
			return
		}
		log.Fatalf(`Error creating PersistenceVolumeClaim "che-data-volume" %s`, err)
	} else {
		log.Infoln(`PersistenceVolumeClaim "che-data-volume" successfully created`)
	}
}

//createCheWorkspacePVC Creates the che workspace Persistence Volume Claim
func (i *InstallerConfig) createCheWorkspacePVC() {
	clientset, err := kubernetes.NewForConfig(i.config)

	if err != nil {
		log.Fatalf("%s", err)
	}

	pvcClient := clientset.CoreV1().PersistentVolumeClaims(i.namespace)

	_, err = createDefaultPVC(pvcClient, "claim-che-workspace")

	if err != nil {
		b := objectExists(objPVC, "claim-che-workspace", err)
		if b {
			log.Infoln(`PersistenceVolumeClaim  "claim-che-workspace" already exists, skipping creation`)
			return
		}
		log.Fatalf(`Error creating PersistenceVolumeClaim "claim-che-workspace" %s`, err)
	} else {
		log.Infoln(`PersistenceVolumeClaim "claim-che-workspace" successfully created`)
	}
}

//createDeploymentConfig Creates the che deploymentconfig
func (i *InstallerConfig) createCheDeploymentConfig() {
	appcs, err := appclient.NewForConfig(i.config)

	if err != nil {
		log.Fatalf("%s", err)
	}

	dcClient := appcs.AppsV1().DeploymentConfigs(i.namespace)

	ts := int64(10000)

	_, err = dcClient.Create(&oappv1.DeploymentConfig{ObjectMeta: metav1.ObjectMeta{
		Name: "che",
		Labels: map[string]string{
			"app": "che",
		},
	}, Spec: oappv1.DeploymentConfigSpec{
		Replicas: 1,
		Selector: map[string]string{
			"app": "che",
		},
		Strategy: oappv1.DeploymentStrategy{
			RecreateParams: &oappv1.RecreateDeploymentStrategyParams{
				TimeoutSeconds: &ts,
			},
			Type: "Recreate",
		},
		Template: &kapi.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"app": "che",
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
			ServiceAccountName: "che",
			Volumes: []kapi.Volume{
				{
					Name: "che-data-volume",
					VolumeSource: kapi.VolumeSource{
						PersistentVolumeClaim: &kapi.PersistentVolumeClaimVolumeSource{
							ClaimName: "che-data-volume",
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
						Name:      "che-server:latest",
					},
				},
			},
			oappv1.DeploymentTriggerPolicy{
				Type: oappv1.DeploymentTriggerOnConfigChange,
			},
		},
	}})

	if err != nil {
		b := objectExists(objDeploymentConfigs, "che", err)
		if b {
			log.Infoln(`DeploymentConfig  "che" already exists, skipping creation`)
			return
		}
		log.Fatalf(`Error creating DeploymentConfig "che" %s`, err)
	} else {
		log.Infoln(`DeploymentConfig "che" successfully created`)
	}
}

//createCheRoute Creates the che route
func (i *InstallerConfig) createCheRoute() {

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
	}, Spec: i.OpenShiftType.routeSpec()})

	if err != nil {
		b := objectExists(objRoute, "che", err)
		if b {
			log.Infoln(`Route "che" already exists, skipping creation`)
			return
		}
		log.Fatalf(`Error creating Route "che" %s`, err)
	} else {
		log.Infoln(`Route "che" successfully created`)
	}
}

//createCheService Creates the che service
func (i *InstallerConfig) createCheService() {
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
			log.Infoln(`Service "che-host" already exists, skipping creation`)
			return
		}
		log.Fatalf(`Error creating Service "che" %s`, err)
	} else {
		log.Infoln(`Service "che-host" successfully created`)
	}
}

//createCheServiceAccount Creates the che service account
func (i *InstallerConfig) createCheServiceAccount() {

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
			log.Infoln(`ServiceAccount "che" already exists, skipping creation`)
			return
		}
		log.Fatalf(`Error creating ServiceAccount "che" %s`, err)
	} else {
		log.Infoln(`ServiceAccount "che" successfully created`)
	}
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

//objectExists simple method to check if the object exists or not
func objectExists(object, name string, err error) bool {
	return strings.Contains(err.Error(), fmt.Sprintf(`%s "%s" already exists`, object, name))
}

// build the Che configmap based on the OpenShiftType
func (ot OpenShiftType) configMap(project, domain string) map[string]string {
	cm := map[string]string{
		"hostname-http":                                                  fmt.Sprintf(`%s-che.%s`, project, domain),
		"workspace-storage":                                              "/home/user/che/workspaces",
		"workspace-storage-create-folders":                               "false",
		"local-conf-dir":                                                 "/etc/conf",
		"openshift-serviceaccountname":                                   "che",
		"che-server-evaluation-strategy":                                 "docker-local-custom",
		"che.logs.dir":                                                   "/data/logs",
		"che.docker.server_evaluation_strategy.custom.template":          "<serverName>-<if(isDevMachine)><workspaceIdWithoutPrefix><else><machineName><endif>-<externalAddress>",
		"che.docker.server_evaluation_strategy.custom.external.protocol": "https",
		"che.predefined.stacks.reload_on_start":                          "true",
		"log-level":                                                      "INFO",
		"docker-connector":                                               "openshift",
		"port":                                                           "8080",
		"remote-debugging-enabled":                                       "false",
		"che-oauth-github-forceactivation":                               "true",
		"workspaces-memory-limit":                                        "1900Mi",
		"workspaces-memory-request":                                      "1100Mi",
		"enable-workspaces-autostart":                                    "false",
		"che-server-java-opts":                                           "-XX:+UseG1GC -XX:+UseStringDeduplication -XX:MinHeapFreeRatio=20 -XX:MaxHeapFreeRatio=40 -XX:MaxRAM=600m -Xms256m",
		"che-workspaces-java-opts":                                       "-XX:+UseG1GC -XX:+UseStringDeduplication -XX:MinHeapFreeRatio=20 -XX:MaxHeapFreeRatio=40 -XX:MaxRAM=1200m -Xms256m",
		"che-openshift-secure-routes":                                    "true",
		"che-secure-external-urls":                                       "true",
		"che-server-timeout-ms":                                          "3600000",
		"che-openshift-precreate-subpaths":                               "false",
		"che-workspace-auto-snapshot":                                    "false",
	}

	switch ot {
	case "osio":
		cm["keycloak-oso-endpoint"] = "${KEYCLOAK_OSO_ENDPOINT}"
		cm["keycloak-github-endpoint"] = "${KEYCLOAK_GITHUB_ENDPOINT}"
		cm["keycloak-disabled"] = "false"
	case "ocp":
		cm["keycloak-oso-endpoint"] = "${KEYCLOAK_OSO_ENDPOINT}"
		cm["keycloak-github-endpoint"] = "${KEYCLOAK_GITHUB_ENDPOINT}"
		cm["keycloak-disabled"] = "false"
	default:
		cm["che.docker.server_evaluation_strategy.custom.external.protocol"] = "http"
		cm["che.predefined.stacks.reload_on_start"] = "false"
		cm["che-openshift-secure-routes"] = "false"
		cm["che-secure-external-urls"] = "false"
		cm["che-openshift-precreate-subpath"] = "true"
		cm["keycloak-disabled"] = "true"
		cm["workspaces-memory-limit"] = "1300Mi"
		cm["workspaces-memory-request"] = "500Mi"
	}

	log.Debugf("ConfigMap Data: %s", cm)
	return cm
}

//routeSpec builds route spec based on type of OpenShift cluster
func (ot OpenShiftType) routeSpec() ov1.RouteSpec {
	routeSpec := &ov1.RouteSpec{
		To: ov1.RouteTargetReference{
			Kind: "Service",
			Name: "che-host",
		}}

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

//Customized Environment Variables for each OpenShiftType
func (ot OpenShiftType) cheEnvVars() []kapi.EnvVar {
	switch ot {
	case "osio":
		return OSIOCheEnvVars()
	case "ocp":
		return OCPCheEnvVars()
	default:
		log.Infoln("Keycloak will be disabled for default cases like minishift")
		return CheEnvVars()
	}
}
