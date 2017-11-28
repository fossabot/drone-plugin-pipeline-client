package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gopkg.in/go-playground/validator.v9"
	"io"
	"net/http"
	"net/http/httputil"
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
		Id         int        `json:"-,omitempty"`
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
		SpotPrice    float64 `json:"spotPrice,omitempty"`
	}

	Deployment struct {
		Name  string `json:"name" validate:"required"`
		State string `json:"state" validate:"required"`
	}
)

type (
	ClusterResponse struct {
		Message string        `json:"message"`
		Status  int           `json:"status"`
		Data    []ClusterData `json:"data,omitempty"`
	}

	ClusterData struct {
		Id   int
		Name string
		Ip   string
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
		return nil
	}

	Infof("Cluster desired state: %s", p.Config.Cluster.State)
	settingUpClusterId(&p.Config)

	//if cluster exists
	if p.Config.Cluster.State == createdState && clusterExists(&p.Config) == false {
		createCluster(&p.Config)
	} else if p.Config.Cluster.State == createdState {
		Infof("Cluster already present: %s", p.Config.Cluster.Name)
		Infof("Your cluster id: %d", p.Config.Cluster.Id)
	} else if p.Config.Cluster.State == deletedState && clusterExists(&p.Config) == true {
		Infof("Your cluster id: %d", p.Config.Cluster.Id)
		deleteCluster(&p.Config)
	} else if  p.Config.Cluster.State == deletedState {
		Infof("Cluster %s doesn't exists or already deleted, nothing to do ", p.Config.Cluster.Name)
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

func settingUpClusterId(config *Config) {
	url := fmt.Sprintf("%s/clusters", config.Endpoint)
	resp := apiCall(url, "GET", config.Username, config.Password, nil)

	result := ClusterResponse{}
	if resp.StatusCode == 200 {
		err := json.NewDecoder(resp.Body).Decode(&result)

		if err != nil {
			Fatalf("failed to parse /api/v1/clusters to go struct: %+v", resp)
		}
	}

	for _, cluster := range result.Data {
		if cluster.Name == config.Cluster.Name {
			config.Cluster.Id = cluster.Id
		}
	}
}

func deleteCluster(config *Config) bool {
	Infof("Delete %s cluster\n", config.Cluster.Name)
	url := fmt.Sprintf("%s/clusters/%d", config.Endpoint, config.Cluster.Id)
	resp := apiCall(url, "DELETE", config.Username, config.Password, nil)

	if resp.StatusCode == 201 {
		Infof("Cluster (%s) will be deleted", config.Cluster.Name)
		return true
	}

	if resp.StatusCode == 404 {
		Errorf("Unable to delete cluster %s", config.Cluster.Name)
		return false
	}

	Fatalf("Unexpected error %+v", resp)
	return false
}

func createCluster(config *Config) bool {

	Infof("Create %s cluster", config.Cluster.Name)

	url := fmt.Sprintf("%s/clusters", config.Endpoint)
	param, _ := json.Marshal(config.Cluster)
	resp := apiCall(url, "POST", config.Username, config.Password, bytes.NewBuffer(param))

	if resp.StatusCode == 201 {
		Infof("Cluster (%s) will be created", config.Cluster.Name)
		return true
	}

	Fatalf("Unexpected error %+v", resp)
	return false
}

func clusterExists(config *Config) bool {
	if config.Cluster.Id > 0 {
		return true
	}
	return false
}
