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
	"k8s.io/client-go/rest"
	kapi "k8s.io/kubernetes/pkg/api/v1"
)

// OpenShiftType to distinguish between minishift, ocp, osio etc.,
type OpenShiftType string

//InstallerConfig will be used during installation processs
type InstallerConfig struct {
	config        *rest.Config
	namespace     string
	OpenShiftType OpenShiftType
	ImageTag      string
}

//CheEnvVars environment variables required by Che app
func CheEnvVars() []kapi.EnvVar {
	return []kapi.EnvVar{
		{
			Name: "CHE_DOCKER_IP_EXTERNAL",
			ValueFrom: &kapi.EnvVarSource{
				ConfigMapKeyRef: &kapi.ConfigMapKeySelector{
					Key: "hostname-http",
					LocalObjectReference: kapi.LocalObjectReference{
						Name: "che",
					},
				},
			},
		},
		{
			Name: "CHE_WORKSPACE_STORAGE",
			ValueFrom: &kapi.EnvVarSource{
				ConfigMapKeyRef: &kapi.ConfigMapKeySelector{
					Key: "workspace-storage",
					LocalObjectReference: kapi.LocalObjectReference{
						Name: "che",
					},
				},
			},
		},
		{
			Name: "CHE_LOGS_DIR",
			ValueFrom: &kapi.EnvVarSource{
				ConfigMapKeyRef: &kapi.ConfigMapKeySelector{
					Key: "che.logs.dir",
					LocalObjectReference: kapi.LocalObjectReference{
						Name: "che",
					},
				},
			},
		},
		{
			Name: "CHE_WORKSPACE_STORAGE_CREATE_FOLDERS",
			ValueFrom: &kapi.EnvVarSource{
				ConfigMapKeyRef: &kapi.ConfigMapKeySelector{
					Key: "workspace-storage-create-folders",
					LocalObjectReference: kapi.LocalObjectReference{
						Name: "che",
					},
				},
			},
		},
		{
			Name: "CHE_LOCAL_CONF_DIR",
			ValueFrom: &kapi.EnvVarSource{
				ConfigMapKeyRef: &kapi.ConfigMapKeySelector{
					Key: "local-conf-dir",
					LocalObjectReference: kapi.LocalObjectReference{
						Name: "che",
					},
				},
			},
		},
		{
			Name: "CHE_OPENSHIFT_PROJECT",
			ValueFrom: &kapi.EnvVarSource{
				FieldRef: &kapi.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
		{
			Name: "CHE_OPENSHIFT_SERVICEACCOUNTNAME",
			ValueFrom: &kapi.EnvVarSource{
				ConfigMapKeyRef: &kapi.ConfigMapKeySelector{
					Key: "openshift-serviceaccountname",
					LocalObjectReference: kapi.LocalObjectReference{
						Name: "che",
					},
				},
			},
		},
		{
			Name: "CHE_DOCKER_SERVER__EVALUATION__STRATEGY",
			ValueFrom: &kapi.EnvVarSource{
				ConfigMapKeyRef: &kapi.ConfigMapKeySelector{
					Key: "che-server-evaluation-strategy",
					LocalObjectReference: kapi.LocalObjectReference{
						Name: "che",
					},
				},
			},
		},
		{
			Name: "CHE_DOCKER_SERVER__EVALUATION__STRATEGY_CUSTOM_TEMPLATE",
			ValueFrom: &kapi.EnvVarSource{
				ConfigMapKeyRef: &kapi.ConfigMapKeySelector{
					Key: "che.docker.server_evaluation_strategy.custom.template",
					LocalObjectReference: kapi.LocalObjectReference{
						Name: "che",
					},
				},
			},
		},
		{
			Name: "CHE_DOCKER_SERVER__EVALUATION__STRATEGY_CUSTOM_EXTERNAL_PROTOCOL",
			ValueFrom: &kapi.EnvVarSource{
				ConfigMapKeyRef: &kapi.ConfigMapKeySelector{
					Key: "che.docker.server_evaluation_strategy.custom.external.protocol",
					LocalObjectReference: kapi.LocalObjectReference{
						Name: "che",
					},
				},
			},
		},
		{
			Name: "CHE_PREDEFINED_STACKS_RELOAD__ON__START",
			ValueFrom: &kapi.EnvVarSource{
				ConfigMapKeyRef: &kapi.ConfigMapKeySelector{
					Key: "che.predefined.stacks.reload_on_start",
					LocalObjectReference: kapi.LocalObjectReference{
						Name: "che",
					},
				},
			},
		},
		{
			Name: "CHE_LOG_LEVEL",
			ValueFrom: &kapi.EnvVarSource{
				ConfigMapKeyRef: &kapi.ConfigMapKeySelector{
					Key: "log-level",
					LocalObjectReference: kapi.LocalObjectReference{
						Name: "che",
					},
				},
			},
		},
		{
			Name: "CHE_PORT",
			ValueFrom: &kapi.EnvVarSource{
				ConfigMapKeyRef: &kapi.ConfigMapKeySelector{
					Key: "port",
					LocalObjectReference: kapi.LocalObjectReference{
						Name: "che",
					},
				},
			},
		},
		{
			Name: "CHE_DOCKER_CONNECTOR",
			ValueFrom: &kapi.EnvVarSource{
				ConfigMapKeyRef: &kapi.ConfigMapKeySelector{
					Key: "docker-connector",
					LocalObjectReference: kapi.LocalObjectReference{
						Name: "che",
					},
				},
			},
		},
		{
			Name: "CHE_DEBUG_SERVER",
			ValueFrom: &kapi.EnvVarSource{
				ConfigMapKeyRef: &kapi.ConfigMapKeySelector{
					Key: "remote-debugging-enabled",
					LocalObjectReference: kapi.LocalObjectReference{
						Name: "che",
					},
				},
			},
		},
		{
			Name: "CHE_OAUTH_GITHUB_FORCEACTIVATION",
			ValueFrom: &kapi.EnvVarSource{
				ConfigMapKeyRef: &kapi.ConfigMapKeySelector{
					Key: "che-oauth-github-forceactivation",
					LocalObjectReference: kapi.LocalObjectReference{
						Name: "che",
					},
				},
			},
		},
		{
			Name: "CHE_OPENSHIFT_WORKSPACE_MEMORY_OVERRIDE",
			ValueFrom: &kapi.EnvVarSource{
				ConfigMapKeyRef: &kapi.ConfigMapKeySelector{
					Key: "workspaces-memory-limit",
					LocalObjectReference: kapi.LocalObjectReference{
						Name: "che",
					},
				},
			},
		},
		{
			Name: "CHE_OPENSHIFT_WORKSPACE_MEMORY_REQUEST",
			ValueFrom: &kapi.EnvVarSource{
				ConfigMapKeyRef: &kapi.ConfigMapKeySelector{
					Key: "workspaces-memory-request",
					LocalObjectReference: kapi.LocalObjectReference{
						Name: "che",
					},
				},
			},
		},
		{
			Name: "CHE_WORKSPACE_AUTO__START",
			ValueFrom: &kapi.EnvVarSource{
				ConfigMapKeyRef: &kapi.ConfigMapKeySelector{
					Key: "enable-workspaces-autostart",
					LocalObjectReference: kapi.LocalObjectReference{
						Name: "che",
					},
				},
			},
		},
		{
			Name: "JAVA_OPTS",
			ValueFrom: &kapi.EnvVarSource{
				ConfigMapKeyRef: &kapi.ConfigMapKeySelector{
					Key: "che-server-java-opts",
					LocalObjectReference: kapi.LocalObjectReference{
						Name: "che",
					},
				},
			},
		},
		{
			Name: "CHE_WORKSPACE_JAVA_OPTIONS",
			ValueFrom: &kapi.EnvVarSource{
				ConfigMapKeyRef: &kapi.ConfigMapKeySelector{
					Key: "che-workspaces-java-opts",
					LocalObjectReference: kapi.LocalObjectReference{
						Name: "che",
					},
				},
			},
		},
		{
			Name: "CHE_OPENSHIFT_SECURE_ROUTES",
			ValueFrom: &kapi.EnvVarSource{
				ConfigMapKeyRef: &kapi.ConfigMapKeySelector{
					Key: "che-openshift-secure-routes",
					LocalObjectReference: kapi.LocalObjectReference{
						Name: "che",
					},
				},
			},
		},
		{
			Name: "CHE_DOCKER_SERVER__EVALUATION__STRATEGY_SECURE_EXTERNAL_URLS",
			ValueFrom: &kapi.EnvVarSource{
				ConfigMapKeyRef: &kapi.ConfigMapKeySelector{
					Key: "che-secure-external-urls",
					LocalObjectReference: kapi.LocalObjectReference{
						Name: "che",
					},
				},
			},
		},
		{
			Name: "CHE_OPENSHIFT_SERVER_INACTIVE_STOP_TIMEOUT_MS",
			ValueFrom: &kapi.EnvVarSource{
				ConfigMapKeyRef: &kapi.ConfigMapKeySelector{
					Key: "che-server-timeout-ms",
					LocalObjectReference: kapi.LocalObjectReference{
						Name: "che",
					},
				},
			},
		},
		{
			Name: "CHE_OPENSHIFT_PRECREATE_WORKSPACE_DIRS",
			ValueFrom: &kapi.EnvVarSource{
				ConfigMapKeyRef: &kapi.ConfigMapKeySelector{
					Key: "che-openshift-precreate-subpaths",
					LocalObjectReference: kapi.LocalObjectReference{
						Name: "che",
					},
				},
			},
		},
		{
			Name: "CHE_KEYCLOAK_DISABLED",
			ValueFrom: &kapi.EnvVarSource{
				ConfigMapKeyRef: &kapi.ConfigMapKeySelector{
					Key: "keycloak-disabled",
					LocalObjectReference: kapi.LocalObjectReference{
						Name: "che",
					},
				},
			},
		},
		{
			Name: "CHE_WORKSPACE_AUTO__SNAPSHOT",
			ValueFrom: &kapi.EnvVarSource{
				ConfigMapKeyRef: &kapi.ConfigMapKeySelector{
					Key: "che-workspace-auto-snapshot",
					LocalObjectReference: kapi.LocalObjectReference{
						Name: "che",
					},
				},
			},
		},
	}
}

//OCPCheEnvVars additional Env variables that might be needed for Che on OCP
//Separate methods are maintained to allow changes specific to each OpenShift type
func OCPCheEnvVars() []kapi.EnvVar {
	return extendAndCopy()
}

func extendAndCopy() []kapi.EnvVar {
	v := CheEnvVars()
	n := len(v)

	v2 := make([]kapi.EnvVar, n, 2*cap(v))

	copy(v2, v)

	v2[n] = kapi.EnvVar{
		Name: "CHE_KEYCLOAK_OSO_ENDPOINT",
		ValueFrom: &kapi.EnvVarSource{
			ConfigMapKeyRef: &kapi.ConfigMapKeySelector{
				Key: "keycloak-oso-endpoint",
				LocalObjectReference: kapi.LocalObjectReference{
					Name: "che",
				},
			},
		},
	}

	v2[n+1] = kapi.EnvVar{
		Name: "CHE_KEYCLOAK_GITHUB_ENDPOINT",
		ValueFrom: &kapi.EnvVarSource{
			ConfigMapKeyRef: &kapi.ConfigMapKeySelector{
				Key: "keycloak-github-endpoint",
				LocalObjectReference: kapi.LocalObjectReference{
					Name: "che",
				},
			},
		},
	}

	return v2
}
