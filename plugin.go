package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"context"
	"path"

	. "github.com/banzaicloud/banzai-types/components"
	"github.com/pkg/errors"
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
		Repo    Repo
		Build   Build
		Commit  Commit
		Config  Config
		ApiCall ApiCaller
	}

	Config struct {
		Cluster     *CustomCluster
		Deployment  *Deployment
		Endpoint    string
		Token       string
		OrgId       int
		WaitTimeout int64
		ProfileName string
	}

	CustomCluster struct {
		*CreateClusterRequest
		State string
	}

	Deployment struct {
		Name        string                 `json:"name"`
		ReleaseName string                 `json:"release_name"`
		State       string                 `json:"state"`
		Values      map[string]interface{} `json:"values"`
	}

	ConfigResponse struct {
		Status int    `json:"status"`
		Data   string `json:"data,omitempty"`
	}

	ApiCaller func(config *Config, url string, method string, body io.Reader) *http.Response
)

const (
	createdState = "created"
	deletedState = "deleted"
)

var validate *validator.Validate

func (p *Plugin) Exec() error {
	resourceCreationTimeout := time.Duration(p.Config.WaitTimeout) * time.Second
	validate = validator.New()

	err := validate.Struct(p)
	if err != nil {
		for _, v := range err.(validator.ValidationErrors) {
			log.Errorf("[%s] field validation error (%+v)", v.Field(), v)
		}
		return errors.Wrap(err, "validation error(s)")
	}

	orgId, err := p.GetOrgId()
	if err != nil {
		return errors.Wrap(err, "could not retrieve org id")
	}
	log.Debugf("retrieved org id: [ %d ]", orgId)
	log.Infof("cluster desired state: [%s]", p.Config.Cluster.State)

	clusterExists := p.ClusterExists()

	if p.Config.Cluster.State == createdState && !clusterExists {
		_, err := p.createCluster()

		if err != nil {
			log.Fatalf("cluster creation failed: [%s]", err.Error())
			return errors.Wrap(err, "cluster creation failed")
		}

		err = p.waitForResource(resourceCreationTimeout, p.ClusterExists)
		if err != nil {
			log.Error("error while waiting for cluster creation")
			return errors.Wrap(err, "error while waiting for cluster creation")
		}

		log.Infof("cluster [%s] started.", p.Config.Cluster.Name)

	} else if p.Config.Cluster.State == createdState {
		log.Infof("using existing cluster: [%s]", p.Config.Cluster.Name)
	} else if p.Config.Cluster.State == deletedState && clusterExists {
		p.deleteCluster()
		return nil
	} else {
		return nil
	}

	log.Info("setting up helm ...")
	err = p.waitForResource(resourceCreationTimeout, p.isHelmReady)
	if err != nil {
		log.Error("error while waiting for cluster creation")
		return errors.Wrap(err, "error while waiting for cluster creation")
	}
	log.Info("helm is ready.")

	if p.Config.Cluster.State == createdState {
		p.dumpClusterConfig()
	}

	if len(p.Config.Deployment.Name) > 0 {
		log.Infof("checking deployment [%s]", p.Config.Deployment.Name)
		if p.Config.Deployment.State == createdState && !p.DeploymentExists() {
			p.installDeployment()

			err = p.waitForResource(resourceCreationTimeout, p.DeploymentExists)
			if err != nil {
				log.Error("error while waiting for deployment creation")
				return errors.Wrap(err, "error while waiting for deployment creation")
			}

		} else if p.Config.Deployment.State == createdState {
			log.Infof("deployment [%s] already exists, nothing to do", p.Config.Deployment.Name)
		} else if p.Config.Deployment.State == deletedState && p.DeploymentExists() {
			p.deleteDeployment()
		}
	}

	return nil
}

// requestAuth fills the authorization header for the provided request based on the configuration
func (config *Config) requestAuth(request *http.Request) error {
	if request == nil {
		log.Error("http request is nil")
		return errors.New("http request is nil")
	}

	if len(config.Token) > 0 {
		log.Debug("bearer token provided, setting the Authorization header")
		request.Header.Set("Authorization", "Bearer "+config.Token)
		return nil
	}

	log.Info("no credentials provided, no Authorization header is set ")
	return nil
}

func ApiCall(config *Config, url string, method string, body io.Reader) *http.Response {
	log.Debugf("api call args -> url: [%s], method: [%s]", url, method)

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		log.Fatalf("could not create request [%s]", err.Error())
	}

	err = config.requestAuth(req)
	if err != nil {
		log.Fatalf("could not create request [%s]", err.Error())
	}

	if method == http.MethodPost {
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	}

	if err != nil {
		log.Fatalf("failed to build http request: [%s]", err.Error())
	}

	req.Header.Add("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		log.Fatalf("failed to call [%s] on [%s] , error: [%s]", method, url, err.Error())
	}

	return resp
}

func (p *Plugin) deleteCluster() bool {
	log.Infof("initiating delete for cluster [ %s ]", p.Config.Cluster.Name)

	url := fmt.Sprintf("%s/orgs/%d/clusters/%s?field=name", p.Config.Endpoint, p.Config.OrgId, p.Config.Cluster.Name)
	resp := p.ApiCall(&p.Config, url, http.MethodDelete, nil)
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusAccepted {
		log.Infof("cluster [%s] is being deleted", p.Config.Cluster.Name)
		return true
	}

	if resp.StatusCode == http.StatusNotFound {
		log.Infof("cluster [%s] not found", p.Config.Cluster.Name)
		return false
	}

	log.Fatalf("error while deleting cluster %+v", resp)
	return false
}

func (p *Plugin) createCluster() (bool, error) {
	log.Infof("start creating cluster with name: [%s]", p.Config.Cluster.Name)

	url := fmt.Sprintf("%s/orgs/%d/clusters", p.Config.Endpoint, p.Config.OrgId)
	param, err := json.Marshal(p.Config.Cluster)

	if err != nil {
		log.Errorf("could not process cluster details. err: [%s]", err.Error())
		return false, err
	}

	resp := p.ApiCall(&p.Config, url, http.MethodPost, bytes.NewBuffer(param))
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK: // 200
		log.Infof("cluster [%s] is being created", p.Config.Cluster.Name)
		return true, nil
	case http.StatusAccepted: // 202
		log.Infof("cluster creation request for [%s] has been accepted", p.Config.Cluster.Name)
		return true, nil
	case http.StatusBadRequest: // 400
		return false, errors.New(fmt.Sprintf("bad request while creating cluster [%s]", resp.Status))
	default:
		return false, errors.New(fmt.Sprintf("unexpected response status code: [ %d ]", resp.StatusCode))

	}

}

func (p *Plugin) isHelmReady() bool {
	url := fmt.Sprintf("%s/orgs/%d/clusters/%s/deployments?field=name", p.Config.Endpoint, p.Config.OrgId, p.Config.Cluster.Name)
	resp := p.ApiCall(&p.Config, url, http.MethodHead, nil)
	defer resp.Body.Close()
	log.Debugf("checking tiller. received response status code: [%d]", resp.StatusCode)

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

func (p *Plugin) DeploymentExists() bool {

	url := fmt.Sprintf("%s/orgs/%d/clusters/%s/deployments/%s?field=name", p.Config.Endpoint, p.Config.OrgId,
		p.Config.Cluster.Name, p.Config.Deployment.ReleaseName)
	resp := p.ApiCall(&p.Config, url, http.MethodHead, nil)
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK: //200
		log.Debugf("deployment [%s] found", p.Config.Deployment.Name)
		return true
	case http.StatusNotFound: // 404
		log.Debugf("deployment [%s] is not found", p.Config.Deployment.Name)
		return false
	case http.StatusNoContent: //204
		log.Debugf("deployment [%s] is not yet ready", p.Config.Deployment.Name)
		return false
	default:
		log.Debugf("(deployment exists req) ignored response status code [%d] ", resp.StatusCode)
	}

	log.Fatalf("error while checking deployment %+v", resp)
	return false
}

func (p *Plugin) ClusterExists() bool {
	url := fmt.Sprintf("%s/orgs/%d/clusters/%s?field=name", p.Config.Endpoint, p.Config.OrgId, p.Config.Cluster.Name)
	resp := p.ApiCall(&p.Config, url, http.MethodHead, nil)
	defer resp.Body.Close()

	log.Debugf("response status code : [%d] ", resp.StatusCode)
	switch resp.StatusCode {
	case http.StatusOK:
		log.Debugf("cluster [%s] exists.", p.Config.Cluster.Name)
		return true
	case http.StatusNotFound:
		log.Debugf("cluster [%s] not found.", p.Config.Cluster.Name)
		return false
	case http.StatusNoContent:
		log.Debugf("cluster [%s] not yet alive.", p.Config.Cluster.Name)
		return false
	case http.StatusBadRequest:
		log.Debugf("cluster [%s] not yet available.", p.Config.Cluster.Name)
		return false

	default:
		log.Debugf("(cluster status req) ignored response code : [%d] ", resp.StatusCode)
	}

	log.Fatalf("error while checking cluster existence. response code: [ %d ], status message: [ %s ]", resp.StatusCode, resp.Status)
	return false
}

func (p *Plugin) dumpClusterConfig() bool {
	url := fmt.Sprintf("%s/orgs/%d/clusters/%s/config?field=name", p.Config.Endpoint, p.Config.OrgId, p.Config.Cluster.Name)
	resp := p.ApiCall(&p.Config, url, http.MethodGet, nil)
	defer resp.Body.Close()

	result := ConfigResponse{}

	if resp.StatusCode == http.StatusOK {

		err := json.NewDecoder(resp.Body).Decode(&result)

		if err != nil {
			log.Fatalf("error while parsing JSON: [%s]", err.Error())
			return false
		}

		wsConfigDir := path.Join(p.Build.Path, ".kube")
		err = os.MkdirAll(wsConfigDir, 0755)

		if err != nil {
			log.Fatalf("unable to create dir: [%s], error: [%s]", wsConfigDir, err.Error())
			return false
		}

		wsConfigFile := path.Join(wsConfigDir, "config")
		err = ioutil.WriteFile(wsConfigFile, []byte(result.Data), 0666)

		if err != nil {
			log.Fatalf("error while writing config file: [%s], error [%s]", wsConfigFile, err.Error())
			return false
		}

		log.Debugf("export KUBECONFIG=%s", wsConfigFile)
		log.Infof("configuration written to workspace: [%s]", wsConfigFile)

		return true
	}

	log.Fatalf("error while dumping configuration: %+v", resp)
	return false
}

func (p *Plugin) installDeployment() bool {

	log.Infof("installing deployment [%s]", p.Config.Deployment.Name)

	url := fmt.Sprintf("%s/orgs/%d/clusters/%s/deployments?field=name", p.Config.Endpoint, p.Config.OrgId, p.Config.Cluster.Name)
	param, _ := json.Marshal(p.Config.Deployment)

	log.Debugf("install deployment request body: [%s]", param)

	resp := p.ApiCall(&p.Config, url, http.MethodPost, bytes.NewBuffer(param))
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		log.Infof("deployment [%s] is being installed", p.Config.Deployment.Name)
		return true
	}

	log.Fatalf("error while installing deployment: %+v", resp)
	return false
}

func (p *Plugin) deleteDeployment() bool {

	log.Infof("initiating delete for deployment [%s]", p.Config.Deployment.Name)

	url := fmt.Sprintf("%s/orgs/%d/clusters/%s/deployments/%s?field=name", p.Config.Endpoint, p.Config.OrgId,
		p.Config.Cluster.Name, p.Config.Deployment.ReleaseName)
	param, _ := json.Marshal(p.Config.Deployment)

	resp := p.ApiCall(&p.Config, url, http.MethodDelete, bytes.NewBuffer(param))
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		log.Infof("deployment [%s] is being deleted", p.Config.Deployment.Name)
		return true
	}

	log.Fatalf("error while deleting deployment %+v", resp)
	return false
}

// GetOrgId retrieves the identifier of the GitHub organization and sets it into the plugin configuration for further reuse
func (p *Plugin) GetOrgId() (int, error) {

	if p.Config.OrgId != 0 {
		log.Debugf("found cached org id: [ %d ]", p.Config.OrgId)
		return p.Config.OrgId, nil
	}

	log.Debugf("looking up id for org: [ %s ]", p.Repo.Owner)
	url := fmt.Sprintf("%s/orgs?field=name", p.Config.Endpoint)
	httpResp := p.ApiCall(&p.Config, url, http.MethodGet, nil)
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		log.Errorf("could not retrieve organizations. cause: [ %s ]", httpResp.Status)
		return 0, errors.Errorf("could not retrieve organizations. status: [ %s ]", httpResp.Status)
	}

	var (
		orgInfoList []struct {
			Id   int    `json:"id"`
			Name string `json:"name"`
		}
		bodyBytes []byte
		err       error
	)

	bodyBytes, err = ioutil.ReadAll(httpResp.Body)
	if err != nil {
		log.Debugf("could not read response body. error: [ %s ] ", err.Error())
		return 0, err
	}

	// parsing the data we need from the response
	log.Debugf("organizations response: [%s]", string(bodyBytes))
	err = json.Unmarshal(bodyBytes, &orgInfoList)
	if err != nil {
		log.Errorf("could not parse orgs response: [ %s ]", err.Error())
		return 0, errors.Wrapf(err, "could not parse orgs response [ %s ]", string(bodyBytes))
	}

	for _, orgInfo := range orgInfoList {

		if orgInfo.Name == p.Repo.Owner {
			log.Debugf("found org: [ %s ], with id: [ %d ]", orgInfo.Name, orgInfo.Id)
			p.Config.OrgId = orgInfo.Id
			return p.Config.OrgId, nil
		}
	}

	log.Debugf("could not find organization: [%s]", p.Repo.Owner)
	return 0, fmt.Errorf("could not find id for organization: [%s]" + p.Repo.Owner)

}

// waitForResource given a timeout period and a resource checker function this method blocks till the resource becomes available
// or the timeout period is exceeded
func (p *Plugin) waitForResource(timeout time.Duration, resourceChecker func() bool) error {
	log.Info("checking for the resource availability ...")

	// set up a context instance to control timeout and cancel waiting for resources
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// this channel is written when the resource becomes available
	pollerChan := make(chan string)

	poller := func() {

		if resourceChecker() {
			// only write in the channel in case the resource is available
			log.Debug("resource available")
			pollerChan <- "ready"
			close(pollerChan)
		}
		log.Debug("resource not yet available")
	}

	for {
		select {
		case <-ctx.Done():
			log.Error("timeout happened")
			return ctx.Err()
		case <-pollerChan:
			log.Debug("resource available")
			return nil
		default:
			go poller()
			time.Sleep(5 * time.Second)
		}
	}

}
