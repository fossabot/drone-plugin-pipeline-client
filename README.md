
## Pipeline API client plugin for Drone

Pipeline REST API client plugin for Drone. A step in the Pipeline PaaS CI/CD component to provision a Kubernetes cluster or use a managed one.

### Example drone config

.drone.yml

    pipeline:
      cluster:
        image: banzaicloud/pipeline_client:latest

        endpoint: http://[your-host-name-or-ip]/pipeline/api/v1/
        username: admin
        password: *****
        log_level: info
        log_format: text

        cluster_name: "test-cluster"
        cluster_location: "eu-west-1"
        cluster_state: "present"

        node_image: ami-XXXXXXXX
        node_instance_type: m4.xlarge

        master_image: ami-XXXXXXXX
        master_instance_type: m4.xlarge

        deployment_name: "zeppelin-k8s-charts"
        deployment_stat: "present"

## Test container/plugin with drone exec

    drone exec --repo-name hello-world --workspace-path drone-test .drone.yml
    
## Build new docker image

    make docker

## For dev env push .env file

.env

    PLUGIN_CLUSTER_NAME=test-cluster
    PLUGIN_CLUSTER_LOCATION="eu-west-1"
    PLUGIN_CLUSTER_STATE="present"

    PLUGIN_NODE_IMAGE=ami-34b6764d
    PLUGIN_NODE_INSTANCE_TYPE=m4.xlarge

    PLUGIN_MASTER_IMAGE=ami-34b6764d
    PLUGIN_MASTER_INSTANCE_TYPE=m4.xlarge

    PLUGIN_ENDPOINT=http://[your-host-name-or-ip]/pipeline/api/v1/
    PLUGIN_USERNAME=admin
    PLUGIN_PASSWORD=***
    PLUGIN_LOG_LEVEL=debug
    PLUGIN_DEPLOYMENT_NAME="zeppelin-k8s-charts"
    PLUGIN_DEPLOYMENT_STATE="present"

### Test with `go run`

    go run -ldflags "-X main.version=1.0" main.go plugin.go log.go --plugin.log.level debug --plugin.log.format text
