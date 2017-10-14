package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

const (
	maxRetries     = 12
	retrySleepTime = 10
	cheEndpointURI = "http://che-myproject.192.168.64.11.nip.io/api"
	newStackURL    = "https://raw.githubusercontent.com/redhat-developer/rh-che/master/assembly/fabric8-stacks/src/main/resources/stacks.json"
)

// Stack represents the id and name of the stack
type stack struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

//addNewStack will add a new Che Stack Json
func addNewStack(stack json.RawMessage) (int, error) {
	req, err := http.NewRequest(http.MethodPost, cheEndpointURI+"/stack/", bytes.NewBuffer(stack))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	if err != nil {
		log.Fatal(err)
	}

	cheClient := http.Client{}
	resp, err := cheClient.Do(req)

	return resp.StatusCode, err
}

//deleteStack will add delete a stack identified by stackID
func deleteStack(stack stack) (int, error) {
	req, err := http.NewRequest(http.MethodDelete, cheEndpointURI+"/stack/"+stack.ID, nil)
	req.Header.Set("Accept", "application/json")

	if err != nil {
		log.Fatal(err)
	}

	cheClient := http.Client{}
	resp, err := cheClient.Do(req)

	return resp.StatusCode, err
}

//newStacks fetches new stacks from remote url
func newStacks() ([]json.RawMessage, error) {
	req, err := http.NewRequest(http.MethodGet, newStackURL, nil)
	req.Header.Set("Accept", "application/json")

	if err != nil {
		log.Fatal(err)
	}

	cheClient := http.Client{}

	resp, err := cheClient.Do(req)

	var newStacks []json.RawMessage

	stackJSON, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Fatal(err)
	}

	json.Unmarshal(stackJSON, &newStacks)

	return newStacks, err
}

//queryStacks will query the Che for existing stack
func queryStacks() ([]stack, error) {
	req, err := http.NewRequest(http.MethodGet, cheEndpointURI+"/stack", nil)
	req.Header.Set("Accept", "application/json")

	if err != nil {
		log.Fatal(err)
	}

	cheClient := http.Client{}

	resp, err := cheClient.Do(req)

	stackJSON, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Fatal(err)
	}

	var stacksIds = make([]stack, 0)

	json.Unmarshal(stackJSON, &stacksIds)

	return stacksIds, err
}

//retryUntilCheIsUp is simple retry function that will keep trying the callback function until all
//retries are exhausted, for each retry it will sleep for sleep seconds
func retryUntilCheIsUp(retries int, sleep time.Duration, callback func() error) error {

	err := callback()

	if err == nil {
		return nil
	}

	if retries--; retries > 0 {
		time.Sleep(sleep * time.Second)
		log.Println("Retrying after err:", err)
		return retryUntilCheIsUp(retries, 10, callback)
	}

	return fmt.Errorf("After %d seconds, last error: %s", sleep, err)
}

func main() {

	err := retryUntilCheIsUp(maxRetries, retrySleepTime, func() error {

		resp, err := http.Get(cheEndpointURI)

		if err != nil {
			return err
		}

		if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusBadRequest {
			return nil
		}

		return fmt.Errorf("Server Error: %v", resp.StatusCode)
	})

	if err != nil {
		log.Fatal(err)
	} else {
		log.Println("Refreshing Stacks")
		result, err := queryStacks()

		if err != nil {
			log.Fatal(err)
		}

		resultCount := len(result)

		for i := 0; i < resultCount; i++ {
			oldStack := result[i]

			status, err := deleteStack(result[i])

			if err != nil {
				log.Fatal(err)
			}

			if status == http.StatusNoContent {
				fmt.Printf("Deleted Old Stack: %s \n", oldStack.Name)
			}
		}
		if resultCount <= 0 {
			fmt.Println("No old Stacks exist")
		}
		newStacks, err := newStacks()

		if err != nil {
			log.Fatal(err)
		}

		for i := 0; i < len(newStacks); i++ {

			bStack := newStacks[i]
			status, err := addNewStack(bStack)

			if err != nil {
				log.Fatal(err)
			}
			var stack stack
			json.Unmarshal(bStack, &stack)
			if status == http.StatusCreated {
				fmt.Printf("Successfully added new stack :%s \n", stack.Name)
			}
		}

	}
}
