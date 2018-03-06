package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"os"
	"time"

	. "github.com/banzaicloud/banzai-types/components"
	log "github.com/sirupsen/logrus"
	"gopkg.in/go-playground/validator.v9"
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
		Token      string
	}

	CustomCluster struct {
		*CreateClusterRequest
		State string
	}

	Deployment struct {
		Name        string                 `json:"name"`
		ReleaseName string                 `json:"releasename"`
		State       string                 `json:"state"`
		Values      map[string]interface{} `json:"values"`
	}
)

type (
	ConfigResponse struct {
		Status int    `json:"status"`
		Data   string `json:"data,omitempty"`
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
			log.Errorf("[%s] field validation error (%+v)", v.Field(), v)
		}
		return errors.New("validation error(s)")
	}

	log.Infof("cluster desired state: [%s]", p.Config.Cluster.State)

	if p.Config.Cluster.State == createdState && !clusterExists(&p.Config) {
		_, err := createCluster(&p.Config)

		if err != nil {
			log.Fatalf("cluster creation failed: %s", err)
			os.Exit(1)
		}

		log.Infof("waiting for the cluster to start ...")

		for ok := true; ok; ok = !clusterExists(&p.Config) {
			ok = !clusterExists(&p.Config)
			if ok {
				time.Sleep(5 * time.Second)
			}
		}
		log.Infof("cluster started.")

	} else if p.Config.Cluster.State == createdState {
		log.Infof("using existing cluster, nothing to do")
	} else if p.Config.Cluster.State == deletedState && clusterExists(&p.Config) {
		deleteCluster(&p.Config)
		return nil
	} else {
		return nil
	}

	if p.Config.Cluster.State == createdState {
		dumpClusterConfig(p)
	}

	log.Infof("setting up helm ...")
	for ok := true; ok; ok = !isHelmReady(&p.Config) {
		ok = !isHelmReady(&p.Config)
		if ok {
			time.Sleep(5 * time.Second)
		}
	}
	log.Infof("helm is ready.")

	if len(p.Config.Deployment.Name) > 0 {
		log.Infof("checking deployment [%s]", p.Config.Deployment.Name)
		if p.Config.Deployment.State == createdState && !deploymentExists(&p.Config) {
			installDeployment(&p.Config)

			for ok := true; ok; ok = !deploymentExists(&p.Config) {
				ok = !deploymentExists(&p.Config)
				if ok {
					time.Sleep(5 * time.Second)
				}
			}
		} else if p.Config.Deployment.State == createdState {
			log.Infof("deployment [%s] already exists, nothing to do", p.Config.Deployment.Name)
		} else if p.Config.Deployment.State == deletedState && deploymentExists(&p.Config) {
			deleteDeployment(&p.Config)
		}
	}

	return nil
}

// requestAuth fills the authorization header for the provided request based on the configuration
func (config *Config) requestAuth(request *http.Request) error {
	if request == nil {
		log.Fatalf("http request is nil")
	}
	if len(config.Token) > 0 {
		log.Debugf("bearer token provided, setting the Authorization header")
		request.Header.Set("Authorization", "Bearer "+config.Token)
		return nil
	}

	if len(config.Username) > 0 {
		log.Debugf("username provided, proceeding to basic auth")
		request.SetBasicAuth(config.Username, config.Password)
		return nil
	}

	log.Infof("no credentials provided, no Authorization header is set ")
	return nil
}

func (config *Config) apiCall(url string, method string, body io.Reader) *http.Response {
	log.Debugf("api call args -> url: [%s], method: [%s]", url, method)
	req, err := http.NewRequest(method, url, body)

	if err != nil {
		log.Fatalf("could not create request [%s]", err)
	}

	config.requestAuth(req)

	if method == http.MethodPost {
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	}

	if err != nil {
		log.Fatalf("failed to build http request: %v", err)
	}

	req.Header.Add("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		log.Fatalf("failed to call [%s] on [%s]: %+v", method, url, err)
	}

	debugReq, _ := httputil.DumpRequest(req, true)
	log.Debugf("Request %s", debugReq)
	debugResp, _ := httputil.DumpResponse(resp, true)
	log.Debugf("Response %s", debugResp)

	defer resp.Body.Close()
	return resp
}

func deleteCluster(config *Config) bool {
	log.Infof("Trying to delete %s cluster\n", config.Cluster.Name)

	url := fmt.Sprintf("%s/clusters/%s?field=name", config.Endpoint, config.Cluster.Name)
	resp := config.apiCall(url, http.MethodDelete, nil)

	if resp.StatusCode == 202 {
		log.Infof("Cluster will be deleted")
		return true
	}

	if resp.StatusCode == 404 {
		log.Errorf("Unable to delete cluster")
		return false
	}

	log.Fatalf("Unexpected error %+v", resp)
	return false
}

func createCluster(config *Config) (bool, error) {
	log.Infof("start creating cluster with name: [%s]", config.Cluster.Name)

	url := fmt.Sprintf("%s/clusters", config.Endpoint)
	param, err := json.Marshal(config.Cluster)

	if err != nil {
		log.Errorf("could not process cluster details. err: %s", err)
		return false, err
	}

	resp := config.apiCall(url, http.MethodPost, bytes.NewBuffer(param))

	switch resp.StatusCode {
	case http.StatusOK: // 200
		log.Infof("cluster [%s] is being created", config.Cluster.Name)
		return true, nil
	case http.StatusBadRequest: // 400
		return false, errors.New(fmt.Sprintf("bad request while creating cluster [%s]", resp.Status))
	default:
		return false, errors.New(fmt.Sprintf("unexpected error %+v", resp))

	}

}

func isHelmReady(config *Config) bool {
	url := fmt.Sprintf("%s/clusters/%s/deployments?field=name", config.Endpoint, config.Cluster.Name)
	resp := config.apiCall(url, http.MethodHead, nil)
	log.Debugf("checking tiller. received response status code: [%s]", resp.StatusCode)
	
	switch resp.StatusCode {
	case http.StatusOK:
		log.Debugf("helm is ready ...")
		return true
	case http.StatusServiceUnavailable:
		log.Debugf("helm is unavailable...")
		return false
	case http.StatusBadRequest:
		// todo fix the api to return the proper statuscode!
		log.Debugf("helm is unavailable ...")
		return false
	default:
		log.Debugf("(helm ready req) ignored status code: [%d]", resp.StatusCode)
	}

	log.Fatalf("Unexpected error %+v", resp)
	return false
}

func deploymentExists(config *Config) bool {
	url := fmt.Sprintf("%s/clusters/%s/deployments/%s?field=name", config.Endpoint, config.Cluster.Name, config.Deployment.ReleaseName)
	resp := config.apiCall(url, http.MethodHead, nil)

	switch resp.StatusCode {
	case http.StatusOK: //200
		log.Debugf("deployment [%s] found", config.Deployment.Name)
		return true
	case http.StatusNotFound: // 404
		log.Debugf("deployment [%s] is not found", config.Deployment.Name)
		return false
	case http.StatusNoContent: //204
		log.Debugf("deployment [%s] is not yet ready", config.Deployment.Name)
		return false
	default:
		log.Debugf("(deployment exists req) ignored response status code [%d] ", resp.StatusCode)
	}

	log.Fatalf("Unexpected error %+v", resp)
	return false
}

func clusterExists(config *Config) bool {
	url := fmt.Sprintf("%s/clusters/%s?field=name", config.Endpoint, config.Cluster.Name)
	resp := config.apiCall(url, http.MethodHead, nil)

	log.Debugf("response status code : [%d] ", resp.StatusCode)
	switch resp.StatusCode {
	case http.StatusOK:
		log.Debugf("cluster [%s] exists.", config.Cluster.Name)
		return true
	case http.StatusNotFound:
		log.Debugf("cluster [%s] not found.", config.Cluster.Name)
		return false
	case http.StatusNoContent:
		log.Debugf("cluster [%s] not yet alive.", config.Cluster.Name)
		return false
	default:
		log.Debugf("(cluster status req) ignored response code : [%s] ", resp.StatusCode)
	}

	log.Fatalf("Unexpected error %+v", resp)
	return false
}

func dumpClusterConfig(plugin *Plugin) bool {
	config := plugin.Config
	build := plugin.Build
	url := fmt.Sprintf("%s/clusters/%s/config?field=name", config.Endpoint, config.Cluster.Name)
	resp := config.apiCall(url, http.MethodGet, nil)

	defer resp.Body.Close()

	result := ConfigResponse{}

	if resp.StatusCode == http.StatusOK {

		err := json.NewDecoder(resp.Body).Decode(&result)

		if err != nil {
			log.Fatalf("Json parse error: [%s]", err)
			return false
		}

		err = os.MkdirAll(build.Path+"/.kube/", 0755)

		if err != nil {
			log.Fatalf("Unable to create .kube dir: %s", err)
			return false
		}

		err = ioutil.WriteFile(build.Path+"/.kube/config", []byte(result.Data), 0666)

		if err != nil {
			log.Fatalf("File write error: %s", err)
			return false
		}

		log.Debugf("export KUBECONFIG=%s", build.Path+"/.kube/config")
		log.Infof("Write .kube/config to workspace")

		return true
	}

	log.Fatalf("Unexpected error %+v", resp)
	return false
}

func installDeployment(config *Config) bool {

	log.Infof("installing deployment [%s]", config.Deployment.Name)

	url := fmt.Sprintf("%s/clusters/%s/deployments?field=name", config.Endpoint, config.Cluster.Name)
	param, _ := json.Marshal(config.Deployment)

	resp := config.apiCall(url, http.MethodPost, bytes.NewBuffer(param))

	if resp.StatusCode == http.StatusCreated {
		log.Infof("deployment [%s] is being installed", config.Deployment.Name)
		return true
	}

	log.Fatalf("Unexpected error %+v", resp)
	return false
}

func deleteDeployment(config *Config) bool {

	log.Infof("deleting deployment [%s]", config.Deployment.Name)

	url := fmt.Sprintf("%s/clusters/%s/deployments/%s?field=name", config.Endpoint, config.Cluster.Name, config.Deployment.ReleaseName)
	param, _ := json.Marshal(config.Deployment)

	resp := config.apiCall(url, http.MethodDelete, bytes.NewBuffer(param))

	if resp.StatusCode == http.StatusOK {
		log.Infof("Deployment [%s] is being deleted", config.Deployment.Name)
		return true
	}

	log.Fatalf("Unexpected error %+v", resp)
	return false
}
