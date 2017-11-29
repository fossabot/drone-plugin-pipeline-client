
## Pipeline API client plugin for Drone

Pipeline REST API client plugin for Drone. A step in the Pipeline PaaS CI/CD component to provision a Kubernetes cluster or use a managed one.

### Example drone config

.drone.yml

    pipeline:
      cluster:
        image: banzaicloud/pipeline_client:latest

        endpoint: http://[your-host-name-or-ip]/pipeline/api/v1
        username: admin
        password: *****
        log_level: info
        log_format: text

        cluster_name: "demo-cluster1"
        cluster_location: "eu-west-1"
        cluster_state: "created"

        node_image: ami-294ffd50
        node_instance_type: m4.xlarge

        master_image: ami-294ffd50
        master_instance_type: m4.xlarge

        deployment_name: ""
        deployment_release_name: ""
        deployment_state: "created"

## Test container/plugin with drone exec

    drone exec --repo-name hello-world --workspace-path drone-test .drone.yml
    
## Build new docker image

    make docker

## For developers
### Use .env file (example)

    cp .env.example .env
    source .env

### Test with `go run`

    go run -ldflags "-X main.version=1.0" main.go plugin.go log.go --plugin.log.level debug --plugin.log.format text
