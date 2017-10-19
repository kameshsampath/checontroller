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
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

//NewRefresher helps in refreshing Che Stacks
func NewRefresher(cheEndpointURI string, newStackURL string) *Config {
	return &Config{
		CheEndpointURI: cheEndpointURI,
		NewStackURL:    newStackURL,
	}
}

//AddNewStack will add a new Che Stack Json
func (c *Config) AddNewStack(stack json.RawMessage) (int, error) {
	req, err := http.NewRequest(http.MethodPost, c.CheEndpointURI+"/api/stack/", bytes.NewBuffer(stack))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	if err != nil {
		return -1, fmt.Errorf("%s", err)
	}

	cheClient := http.Client{}
	resp, err := cheClient.Do(req)

	return resp.StatusCode, err
}

//Delete a Stack based on the stack ID
func (stack *Stack) Delete(c *Config) (int, error) {
	req, err := http.NewRequest(http.MethodDelete, c.CheEndpointURI+"/api/stack/"+stack.ID, nil)
	req.Header.Set("Accept", "application/json")

	if err != nil {
		return -1, fmt.Errorf("%s", err)
	}

	cheClient := http.Client{}
	resp, err := cheClient.Do(req)

	return resp.StatusCode, err
}

//NewStacks fetches new stacks from remote url
func (c *Config) NewStacks() ([]json.RawMessage, error) {
	req, err := http.NewRequest(http.MethodGet, c.NewStackURL, nil)
	req.Header.Set("Accept", "application/json")

	if err != nil {
		return nil, fmt.Errorf("%s", err)
	}

	cheClient := http.Client{}

	resp, err := cheClient.Do(req)

	var newStacks []json.RawMessage

	stackJSON, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, fmt.Errorf("%s", err)
	}

	json.Unmarshal(stackJSON, &newStacks)

	return newStacks, err
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

	if err == nil && resp != nil {
		stackJSON, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			return make([]Stack, 0), fmt.Errorf("%s", err)
		}

		var stacksIds = make([]Stack, 0)

		json.Unmarshal(stackJSON, &stacksIds)

		return stacksIds, err
	}

	return make([]Stack, 0), fmt.Errorf("Response Body is Empty")
}

//RefreshStacks does refreshing of stacks once ChePod is up
func (c *Config) RefreshStacks() {

	log.Infoln("Refreshing Stacks")

	result, err := c.QueryStacks()

	if err != nil {
		log.Errorf("%s", err)
		return
	}

	resultCount := len(result)

	for i := 0; i < resultCount; i++ {
		oldStack := result[i]

		status, err := oldStack.Delete(c)

		if err != nil {
			log.Errorf("%s", err)
		}

		if status == http.StatusNoContent {
			log.Infof("Deleted Old Stack: %s", oldStack.Name)
		}
	}
	if resultCount <= 0 {
		log.Infoln("No old Stacks exist")
	}

	newStacks, err := c.NewStacks()

	if err != nil {
		log.Errorf("%s", err)
	}

	for i := 0; i < len(newStacks); i++ {
		bStack := newStacks[i]
		status, err := c.AddNewStack(bStack)
		if err != nil {
			log.Errorf(" %s \n", err)
		}
		var stack Stack
		json.Unmarshal(bStack, &stack)
		if status == http.StatusCreated {
			log.Infof("Successfully added new stack :%s \n", stack.Name)
		}
	}
}

//TODO not sure we need this, remove it once done
//retryUntilCheIsUp is simple retry function that will keep trying the callback function until all
//retries are exhausted, for each retry it will sleep for sleep seconds
func retryUntilCheIsUp(retries int, sleep int, callback func() error) error {

	err := callback()

	if err == nil {
		return nil
	}

	if retries--; retries > 0 {
		time.Sleep(time.Duration(sleep) * time.Second)
		log.Println("Retrying after err:", err)
		return retryUntilCheIsUp(retries, sleep, callback)
	}

	log.Errorf("After %d retries, last error: %s", retries, err)

	return nil
}
