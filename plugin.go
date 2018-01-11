package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	. "github.com/banzaicloud/banzai-types/components"
	. "github.com/banzaicloud/banzai-types/utils"
	"gopkg.in/go-playground/validator.v9"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"os"
	"time"
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
		Cluster    *CustomCluster
		Deployment *Deployment
		Username   string
		Password   string
		Endpoint   string
	}

	CustomCluster struct {
		*CreateClusterRequest
		State string
	}

	Deployment struct {
		Name        string `json:"name"`
		ReleaseName string `json:"releasename"`
		State       string `json:"state"`
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
			LogErrorf(LOGTAG, "[%s] field validation error (%+v)", v.Field(), v)
		}
		return errors.New("Validation error(s)")
	}

	LogInfof(LOGTAG, "Cluster desired state: %s", p.Config.Cluster.State)

	if p.Config.Cluster.State == createdState && !clusterIsExists(&p.Config) {
		result, err := createCluster(&p.Config)

		if !result {
			LogFatalf(LOGTAG, "Failed: %s", err)
			os.Exit(1)
		}

		for ok := true; ok; ok = !clusterIsExists(&p.Config) {
			ok = !clusterIsExists(&p.Config)
			LogInfo(LOGTAG, "Waiting...")
			time.Sleep(5 * time.Second)
		}
	} else if p.Config.Cluster.State == createdState {
		LogInfo(LOGTAG, "Use existing cluster, nothing to do")
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
		if ok {
			LogInfo(LOGTAG, "Waiting helm ...")
			time.Sleep(5 * time.Second)
		}
	}
	LogDebugf("s", "", p.Config.Deployment.State)
	if p.Config.Deployment.State == createdState && !deploymentIsExists(&p.Config) {
		installDeployment(&p.Config)

		for ok := true; ok; ok = !deploymentIsExists(&p.Config) {
			ok = !deploymentIsExists(&p.Config)
			if ok {
				LogInfo(LOGTAG, "Waiting deployment ...")
				time.Sleep(5 * time.Second)
			}
		}
	} else if p.Config.Deployment.State == createdState {
		LogInfo(LOGTAG, "Use existing deployment, nothing to do")
	} else if p.Config.Deployment.State == deletedState && deploymentIsExists(&p.Config) {
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
		LogFatalf(LOGTAG, "failed to build http request: %v", err)
	}

	req.SetBasicAuth(username, password)
	req.Header.Add("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		LogFatalf(LOGTAG, "failed to call \"%s\" on %s: %+v", method, url, err)
	}

	debugReq, _ := httputil.DumpRequest(req, true)
	LogDebugf(LOGTAG, "Request %s", debugReq)
	debugResp, _ := httputil.DumpResponse(resp, true)
	LogDebugf(LOGTAG, "Response %s", debugResp)

	defer resp.Body.Close()
	return resp
}

func deleteCluster(config *Config) bool {
	LogInfof(LOGTAG, "Trying to delete %s cluster\n", config.Cluster.Name)

	url := fmt.Sprintf("%s/clusters/%s?field=name", config.Endpoint, config.Cluster.Name)
	resp := apiCall(url, "DELETE", config.Username, config.Password, nil)

	if resp.StatusCode == 201 {
		LogInfo(LOGTAG, "Cluster will be deleted")
		return true
	}

	if resp.StatusCode == 404 {
		LogError(LOGTAG, "Unable to delete cluster")
		return false
	}

	LogFatalf(LOGTAG, "Unexpected error %+v", resp)
	return false
}

func createCluster(config *Config) (bool, string) {

	LogInfof(LOGTAG, "Trying create %s cluster", config.Cluster.Name)

	url := fmt.Sprintf("%s/clusters", config.Endpoint)
	param, _ := json.Marshal(config.Cluster)
	resp := apiCall(url, "POST", config.Username, config.Password, bytes.NewBuffer(param))

	if resp.StatusCode == 201 {
		LogInfof(LOGTAG, "Cluster (%s) will be created", config.Cluster.Name)
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
		LogInfo(LOGTAG, "Helm not ready.")
		return false
	}

	LogFatalf(LOGTAG, "Unexpected error %+v", resp)
	return false
}

func deploymentIsExists(config *Config) bool {
	url := fmt.Sprintf("%s/clusters/%s/deployments/%s?field=name", config.Endpoint, config.Cluster.Name, config.Deployment.ReleaseName)
	resp := apiCall(url, "HEAD", config.Username, config.Password, nil)

	if resp.StatusCode == 200 {
		return true
	} else if resp.StatusCode == 404 {
		LogInfo(LOGTAG, "Deployment not found.")
		return false
	} else if resp.StatusCode == 204 {
		LogInfo(LOGTAG, "Deployment isn't ready yet.")
		return false
	}

	LogFatalf(LOGTAG, "Unexpected error %+v", resp)
	return false
}

func clusterIsExists(config *Config) bool {
	url := fmt.Sprintf("%s/clusters/%s?field=name", config.Endpoint, config.Cluster.Name)
	resp := apiCall(url, "HEAD", config.Username, config.Password, nil)

	if resp.StatusCode == 200 {
		return true
	} else if resp.StatusCode == 404 {
		LogInfo(LOGTAG, "Cluster not found.")
		return false
	} else if resp.StatusCode == 204 {
		LogInfo(LOGTAG, "Cluster isn't alive yet.")
		return false
	}

	LogFatalf(LOGTAG, "Unexpected error %+v", resp)
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
			LogFatalf(LOGTAG, "Json parse error %s", err)
			return false
		}

		data, err := base64.StdEncoding.DecodeString(result.Config)

		if err != nil {
			LogFatalf(LOGTAG, "Decoding error %s", err)
			return false
		}

		err = os.MkdirAll(build.Path+"/.kube/", 0755)

		if err != nil {
			LogFatalf(LOGTAG, "Unable to create .kube dir: %s", err)
			return false
		}

		err = ioutil.WriteFile(build.Path+"/.kube/config", data, 0666)

		if err != nil {
			LogFatalf(LOGTAG, "File write error: %s", err)
			return false
		}

		LogDebugf(LOGTAG, "export KUBECONFIG=%s", build.Path+"/.kube/config")
		LogInfo(LOGTAG, "Write .kube/config to workspace")

		return true
	}

	LogFatalf(LOGTAG, "Unexpected error %+v", resp)
	return false
}

func installDeployment(config *Config) bool {

	LogInfof(LOGTAG, "Install %s deployment", config.Deployment.Name)

	url := fmt.Sprintf("%s/clusters/%s/deployments?field=name", config.Endpoint, config.Cluster.Name)
	param, _ := json.Marshal(config.Deployment)

	resp := apiCall(url, "POST", config.Username, config.Password, bytes.NewBuffer(param))

	if resp.StatusCode == 201 {
		LogInfof(LOGTAG, "Deployment (%s) will be installed", config.Deployment.Name)
		return true
	}

	LogFatalf(LOGTAG, "Unexpected error %+v", resp)
	return false
}

func deleteDeployment(config *Config) bool {

	LogInfof(LOGTAG, "Delete %s deployment", config.Deployment.Name)

	url := fmt.Sprintf("%s/clusters/%s/deployments/%s?field=name", config.Endpoint, config.Cluster.Name, config.Deployment.ReleaseName)
	param, _ := json.Marshal(config.Deployment)

	resp := apiCall(url, "DELETE", config.Username, config.Password, bytes.NewBuffer(param))

	if resp.StatusCode == 200 {
		LogInfof(LOGTAG, "Deployment (%s) will be deleted", config.Deployment.Name)
		return true
	}

	LogFatalf(LOGTAG, "Unexpected error %+v", resp)
	return false
}
