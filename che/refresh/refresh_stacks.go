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
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"strconv"
	"time"

	"github.com/kameshsampath/checontroller/util"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

//KeepAlive represents the cases where CheRefreshController needs to be run in Daemon Mode
//Stopped by CTRL + c or some other mode
func KeepAlive(c *Controller) {
	stopCh := make(chan struct{})
	go util.HandleSigterm(stopCh)
	go c.Run(1, stopCh)
	select {}
}

//TickAndRefresh represents a method that runs the CheRefreshController
//Checking every 5 seconds on its status of refresh, once done it quit
func TickAndRefresh(c *Controller) {
	stopCh := make(chan struct{})
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				fmt.Print(".")
				if c.Done {
					close(stopCh)
				}
			case <-stopCh:
				ticker.Stop()
			}
		}
	}()
	go util.HandleSigterm(stopCh)
	c.Run(1, stopCh)

}

//AddNewStack will add a new Che Stack Json
func (c *Config) AddNewStack(stack string) (int, error) {
	req, err := http.NewRequest(http.MethodPost, c.CheEndpointURI+"/api/stack/", bytes.NewBuffer([]byte(stack)))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	if err != nil {
		return -1, fmt.Errorf("%s", err)
	}

	cheClient := http.Client{}
	resp, err := cheClient.Do(req)

	log.Debugf("Response : %#v", resp)

	if resp != nil && err == nil {
		return resp.StatusCode, err
	} else {
		// TODO any other right error code can be sent ?
		return 503, err
	}
}

//Delete a Stack based on the stack ID
func (stack *Stack) Delete(c *Config) (int, error) {

	log.Debugf("Deleting stack with ID:%s", stack.ID)

	req, err := http.NewRequest(http.MethodDelete, c.CheEndpointURI+"/api/stack/"+stack.ID, nil)
	req.Header.Set("Accept", "application/json")

	if err != nil {
		return -1, fmt.Errorf("%s", err)
	}

	cheClient := http.Client{}
	resp, err := cheClient.Do(req)

	log.Debugf("Response : %#v", resp)

	if err == nil {
		return resp.StatusCode, err
	}

	// TODO any other right error code can be sent ?
	return 503, err
}

//NewStacks fetches new stacks from remote url
func (c *Config) NewStacks() ([]string, error) {
	req, err := http.NewRequest(http.MethodGet, c.NewStackURL, nil)
	req.Header.Set("Accept", "application/json")

	log.Infof("Fetching new stacks from %s", c.NewStackURL)

	if err != nil {
		return make([]string, 0), fmt.Errorf("%s", err)
	}
	cheClient := http.Client{}
	resp, err := cheClient.Do(req)

	log.Debugf("Response : %#v", resp)

	if resp != nil && err == nil {
		stackJSON, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return make([]string, 0), fmt.Errorf("%s", err)
		}

		newJSON := stackJSON
		result := gjson.GetBytes(stackJSON, "#.workspaceConfig.environments.default.machines.dev-machine.agents")

		for i := range result.Array() {
			newJSON, err = sjson.DeleteBytes(newJSON, strconv.Itoa(i)+".workspaceConfig.environments.default.machines.dev-machine.agents.-1")
			if err != nil {
				log.Errorf("%v", err)
			}
		}

		result = gjson.ParseBytes(newJSON)
		newStackArr := make([]string, len(result.Array()))

		for i, v := range result.Array() {
			newStackArr[i] = v.Raw
		}

		return newStackArr, nil

	}

	return make([]string, 0), err
}

//QueryStacks will query the Che for existing stacks
func (c *Config) QueryStacks() ([]Stack, error) {

	req, err := http.NewRequest(http.MethodGet, c.CheEndpointURI+"/api/stack", nil)
	req.Header.Set("Accept", "application/json")

	if err != nil {
		log.Errorf("%s", err)
		return make([]Stack, 0), err
	}

	cheClient := http.Client{}
	resp, err := cheClient.Do(req)

	if err != nil {
		log.Errorf("%s", err)
	}

	log.Debugf("Response : %#v", resp)

	var existingStacks []Stack

	if err == nil && resp != nil {

		log.Infoln("Querying Existing Stacks")

		stackJSON, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return make([]Stack, 0), fmt.Errorf("%s", err)
		}

		stacks := gjson.GetManyBytes(stackJSON, "#.id", "#.name")
		if stacks != nil {
			ids := stacks[0].Array()
			names := stacks[1].Array()
			existingStacks = make([]Stack, len(ids), 2*len(ids))
			for i := range ids {
				existingStacks[i] = Stack{
					ID:   strings.TrimSpace(ids[i].Str),
					Name: strings.TrimSpace(names[i].Str),
				}
				log.Debugf("%#v\n", existingStacks[i])
			}

			return existingStacks, nil
		}

	}

	return existingStacks, fmt.Errorf("Response Body is Empty")

}

//RefreshStacks does refreshing of stacks once ChePod is up
func (c *Config) RefreshStacks() {

	log.Infof("Refreshing Stacks Che Endpoint URI: %s", c.CheEndpointURI)

	result, err := c.QueryStacks()
	if err != nil {
		log.Errorf("%s", err)
		return
	}

	resultCount := len(result)

	if resultCount <= 0 {
		log.Infoln("No old Stacks exist")
	} else {
		log.Infof("%d Old Stacks will be deleted", resultCount)
	}

	for _, s := range result {
		status, err := s.Delete(c)
		if err != nil {
			log.Errorf("%s", err)
		}
		if status == http.StatusNoContent {
			log.Infof("Deleted Old Stack: %s", s.Name)
		}
	}

	result, err = c.QueryStacks()

	if err != nil {
		log.Errorf("%s", err)
		return
	}

	//Very if all old stack is cleared
	currentCount := len(result)

	if currentCount == resultCount {
		log.Warnln("Old Stacks still exists and not deleted ..")
	}

	newStacks, err := c.NewStacks()

	if err != nil {
		log.Errorf("%s", err)
	} else {
		for _, s := range newStacks {
			status, err := c.AddNewStack(s)
			if err != nil {
				log.Errorf(" %s \n", err)
			}
			if status == http.StatusCreated {
				log.Infof("Successfully added new stack :%s", gjson.Parse(s).Get("name"))
			}
		}
	}
}
