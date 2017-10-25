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

package cmd

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kameshsampath/checontroller/util"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	//DefaultNewStackURL holds the external url to fetch the New Stacks from
	DefaultNewStackURL = "https://raw.githubusercontent.com/redhat-developer/rh-che/master/assembly/fabric8-stacks/src/main/resources/stacks.json"
)

var (
	cfgFile string
	//Config holds kubernetes rest client config
	Config *rest.Config
	//Namespace holds the default Namespace to be used during install
	Namespace string
	// RootCmd represents the base command when called without any subcommands
	rootCmd = &cobra.Command{
		Use:   "chectl",
		Short: "chectl is a simple commandline utility for Eclipse Che",
		Long: `The utility helps is performing few of common operations that can be performed on Eclipse Che via CLI.
For example: Install Che on to existing OpenShift Cluster using command chectl install`,
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("%s", err)
	}
}

func init() {

	var kubeconfig *string
	var err error

	home := homedir.HomeDir()

	log.Debugf("Home Dir :%s\n", home)

	kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	flag.Parse()

	Config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)

	if err != nil {
		log.Fatalf("Unable to build config, %v", err)
	}

	Namespace = util.DefaultNamespaceFromConfig(kubeconfig)

	rootCmd.AddCommand(NewInstallCmd())
	rootCmd.AddCommand(NewRefreshCmd())

	//TODO add persistable flags
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.chectl.yaml)")

	log.SetFormatter(&log.TextFormatter{FullTimestamp: true, QuoteEmptyFields: true})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home := homedir.HomeDir()
		// Search config in home directory with name ".chectl" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".chectl")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
