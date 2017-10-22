package util

import (
	"strings"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/clientcmd"
)

//DefaultNamespaceFromConfig detect the namespace from current kuberenetes context
func DefaultNamespaceFromConfig(kubeconfig *string) string {
	config, err := clientcmd.LoadFromFile(*kubeconfig)
	if err != nil {
		log.Errorf("Unable to get NS from context of config %s \n", err)
	}
	return strings.Split(config.CurrentContext, "/")[0]
}
