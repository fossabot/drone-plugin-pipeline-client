package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"gopkg.in/go-playground/validator.v9"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"os"
	"time"
	"errors"
)

type (
	Repo struct {
		Owner   string
		Name    string
		Link    string
		Avatar  string
		Branch  string
		Private bool
		Trusted bool
	}

	Build struct {
		Number   int
		Event    string
		Status   string
		Deploy   string
		Created  int64
		Started  int64
		Finished int64
		Link     string
		Path     string
	}

	Author struct {
		Name   string
		Email  string
		Avatar string
	}

	Commit struct {
		Remote  string
		Sha     string
		Ref     string
		Link    string
		Branch  string
		Message string
		Author  Author
	}

	Plugin struct {
		Repo   Repo
		Build  Build
		Commit Commit
		Config Config
	}

	Config struct {
		Cluster  Cluster
		Username string
		Password string
		Endpoint string
	}

	Cluster struct {
		Name       string     `json:"name" validate:"required"`
		Location   string     `json:"location" validate:"required"`
		State      string     `json:"-" validate:"required"`
		Node       Node       `json:"node"`
		Master     Master     `json:"master"`
		Deployment Deployment `json:"deployment,omitempty"`
	}

	Master struct {
		Image        string `validate:"required"`
		InstanceType string `validate:"required"`
	}

	Node struct {
		Image        string  `json:"image" validate:"required"`
		InstanceType string  `json:"instanceType" validate:"required"`
		MinCount     int     `json:"minCount" validate:"required,gt=0"`
		MaxCount     int     `json:"maxCount" validate:"required,gt=0"`
		SpotPrice    string  `json:"spotPrice,omitempty"`
	}

	Deployment struct {
		Name        string `json:"name" validate:"required"`
		ReleaseName string `json:"releasename" validate:"required"`
		State       string `json:"state" validate:"required"`
	}
)

type (
	ConfigResponse struct {
		Status int    `json:"status"`
		Config string `json:"data,omitempty"`
	}
)

const createdState = "created"
const deletedState = "deleted"

var validate *validator.Validate

func (p *Plugin) Exec() error {
	validate = validator.New()

	err := validate.Struct(p)
	if err != nil {
		for _, v := range err.(validator.ValidationErrors) {
			Errorf("[%s] field validation error (%+v)", v.Field(), v)
		}
		return errors.New("Validation error(s)")
	}

	Infof("Cluster desired state: %s", p.Config.Cluster.State)

	if p.Config.Cluster.State == createdState && !clusterIsExists(&p.Config) {
		result, err := createCluster(&p.Config)

		if !result {
			Fatal("Failed: ", err)
			os.Exit(1)
		}

		for ok := true; ok; ok = !clusterIsExists(&p.Config) {
			ok = !clusterIsExists(&p.Config)
			Info("Waiting...")
			time.Sleep(5 * time.Second)
		}
	} else if p.Config.Cluster.State == createdState {
		Info("Use existing cluster, nothing to do")
	} else if p.Config.Cluster.State == deletedState && clusterIsExists(&p.Config) {
		deleteCluster(&p.Config)
		return nil
	} else {
		return nil
	}

	if p.Config.Cluster.State == createdState {
		dumpClusterConfig(p)
	}

	for ok := true; ok; ok = !helmIsReady(&p.Config) {
		ok = !helmIsReady(&p.Config)
		Info("Waiting helm ...")
		time.Sleep(5 * time.Second)
	}

	if p.Config.Cluster.State == createdState && p.Config.Cluster.Deployment.State == createdState && !deploymentIsExists(&p.Config) {
		installDeployment(&p.Config)

		for ok := true; ok; ok = !deploymentIsExists(&p.Config) {
			ok = !deploymentIsExists(&p.Config)
			Info("Waiting deployment ...")
			time.Sleep(5 * time.Second)
		}
	} else if p.Config.Cluster.Deployment.State == createdState {
		Infof("Use existing deployment, nothing to do")
	} else if p.Config.Cluster.Deployment.State == deletedState && deploymentIsExists(&p.Config) {
		deleteDeployment(&p.Config)
	}

	return nil
}

func apiCall(url string, method string, username string, password string, body io.Reader) *http.Response {
	req, err := http.NewRequest(method, url, body)

	if method == "POST" {
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	}

	if err != nil {
		Fatalf("failed to build http request: %v", err)
	}

	req.SetBasicAuth(username, password)
	req.Header.Add("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		Fatalf("failed to call \"%s\" on %s: %+v", method, url, err)
	}

	debugReq, _ := httputil.DumpRequest(req, true)
	Debugf("Request %s", debugReq)
	debugResp, _ := httputil.DumpResponse(resp, true)
	Debugf("Response %s", debugResp)

	defer resp.Body.Close()
	return resp
}

func deleteCluster(config *Config) bool {
	Infof("Trying to delete %s cluster\n", config.Cluster.Name)

	url := fmt.Sprintf("%s/clusters/%s?field=name", config.Endpoint, config.Cluster.Name)
	resp := apiCall(url, "DELETE", config.Username, config.Password, nil)

	if resp.StatusCode == 201 {
		Infof("Cluster will be deleted")
		return true
	}

	if resp.StatusCode == 404 {
		Errorf("Unable to delete cluster")
		return false
	}

	Fatalf("Unexpected error %+v", resp)
	return false
}

func createCluster(config *Config) (bool, string) {

	Infof("Trying create %s cluster", config.Cluster.Name)

	url := fmt.Sprintf("%s/clusters", config.Endpoint)
	param, _ := json.Marshal(config.Cluster)
	resp := apiCall(url, "POST", config.Username, config.Password, bytes.NewBuffer(param))

	if resp.StatusCode == 201 {
		Infof("Cluster (%s) will be created", config.Cluster.Name)
		return true, ""
	} else if resp.StatusCode == 400 {
		err := fmt.Sprintf("Cluster name already exists")
		return false, err
	}

	err := fmt.Sprintf("Unexpected error %+v", resp)
	return false, err
}

func helmIsReady(config *Config) bool {
	url := fmt.Sprintf("%s/clusters/%s/deployments?field=name", config.Endpoint, config.Cluster.Name)
	resp := apiCall(url, "HEAD", config.Username, config.Password, nil)

	if resp.StatusCode == 200 {
		return true
	} else if resp.StatusCode == 503 {
		Infof("Deployment not found.")
		return false
	}

	Fatalf("Unexpected error %+v", resp)
	return false
}

func deploymentIsExists(config *Config) bool {
	url := fmt.Sprintf("%s/clusters/%s/deployments/%s?field=name", config.Endpoint, config.Cluster.Name, config.Cluster.Deployment.ReleaseName)
	resp := apiCall(url, "HEAD", config.Username, config.Password, nil)

	if resp.StatusCode == 200 {
		return true
	} else if resp.StatusCode == 404 {
		Infof("Deployment not found.")
		return false
	} else if resp.StatusCode == 204 {
		Infof("Deployment isn't ready yet.")
		return false
	}

	Fatalf("Unexpected error %+v", resp)
	return false
}

func clusterIsExists(config *Config) bool {
	url := fmt.Sprintf("%s/clusters/%s?field=name", config.Endpoint, config.Cluster.Name)
	resp := apiCall(url, "HEAD", config.Username, config.Password, nil)

	if resp.StatusCode == 200 {
		return true
	} else if resp.StatusCode == 404 {
		Infof("Cluster not found.")
		return false
	} else if resp.StatusCode == 204 {
		Infof("Cluster isn't alive yet.")
		return false
	}

	Fatalf("Unexpected error %+v", resp)
	return false
}

func dumpClusterConfig(plugin *Plugin) bool {
	config := plugin.Config
	build := plugin.Build
	url := fmt.Sprintf("%s/clusters/%s/config?field=name", config.Endpoint, config.Cluster.Name)
	resp := apiCall(url, "GET", config.Username, config.Password, nil)

	defer resp.Body.Close()

	result := ConfigResponse{}

	if resp.StatusCode == 200 {

		err := json.NewDecoder(resp.Body).Decode(&result)

		if err != nil {
			Fatalf("Json parse error %s", err)
			return false
		}

		data, err := base64.StdEncoding.DecodeString(result.Config)

		if err != nil {
			Fatalf("Decoding error %s", err)
			return false
		}

		err = os.MkdirAll(build.Path+"/.kube/", 0755)

		if err != nil {
			Fatalf("Unable to create .kube dir: %s", err)
			return false
		}

		err = ioutil.WriteFile(build.Path+"/.kube/config", data, 0666)

		if err != nil {
			Fatalf("File write error: %s", err)
			return false
		}

		Debugf("export KUBECONFIG=%s", build.Path+"/.kube/config")
		Infof("Write .kube/config to workspace")

		return true
	}

	Fatalf("Unexpected error %+v", resp)
	return false
}

func installDeployment(config *Config) bool {

	Infof("Install %s deployment", config.Cluster.Deployment.Name)

	url := fmt.Sprintf("%s/clusters/%s/deployments?field=name", config.Endpoint, config.Cluster.Name)
	param, _ := json.Marshal(config.Cluster.Deployment)

	resp := apiCall(url, "POST", config.Username, config.Password, bytes.NewBuffer(param))

	if resp.StatusCode == 201 {
		Infof("Deployment (%s) will be installed", config.Cluster.Deployment.Name)
		return true
	}

	Fatalf("Unexpected error %+v", resp)
	return false
}

func deleteDeployment(config *Config) bool {

	Infof("Delete %s deployment", config.Cluster.Deployment.Name)

	url := fmt.Sprintf("%s/clusters/%s/deployments/%s?field=name", config.Endpoint, config.Cluster.Name, config.Cluster.Deployment.ReleaseName)
	param, _ := json.Marshal(config.Cluster.Deployment)

	resp := apiCall(url, "DELETE", config.Username, config.Password, bytes.NewBuffer(param))

	if resp.StatusCode == 200 {
		Infof("Deployment (%s) will be deleted", config.Cluster.Deployment.Name)
		return true
	}

	Fatalf("Unexpected error %+v", resp)
	return false
}
