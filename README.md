
## Pipeline API client plugin for Drone

Pipeline REST API client plugin for Drone. A step in the Pipeline PaaS CI/CD component to provision a Kubernetes cluster or use a managed one.

### Example drone config

.drone.yml

## Plugin image: 

[banzaicloud/pipeline_client](https://hub.docker.com/r/banzaicloud/plugin-pipeline-client/)    

#### Specified required secrets



### Create or use existing cluster example (Amazon)

    pipeline:
        cluster_name: "demo-cluster1"
        image: banzaicloud/pipeline_client:latest
        secrets: [ plugin_endpoint, plugin_username, plugin_password]

### Create or use existing cluster example (Azure)
    pipeline:
        cluster_name: "demo-cluster1"
        cluster_provider: azure
        image: banzaicloud/pipeline_client:latest
        secrets: [ plugin_endpoint, plugin_username, plugin_password]

### Global options

| Option           | Description             | Default  | Required |
| -------------    | ----------------------- | --------:| --------:|
| cluster_name     | Specified cluster name  | ""       | Yes      |
| cluster_provider | Specified supporter provider (`amazon`, `azure`) | amazon   | No       |
| log_level        | Specified log level (`info`, `warning`,`error`, `critical`) | info   | No       |
| log_format       | Specified log format (`json`, `text`) | json   | No       |

### Provider specific options
#### Amazon
| Option                      | Description              | Default  | Required |
| -------------               | -----------------------  | --------:| --------:|
| amazon_node_type            | Specified cluster name   | ""       | Yes      |
| amazon_master_image         | Specified Image for master node  | ami-06d1667f| No       |
| amazon_master_instance_type | Specified instance type for master node | "m4.xlarge"   | No       |
| amazon_node_image           | Specified Image for node | ami-06d1667f| No       |
| amazon_node_instance_type   | Specified Instance type for node | "m4.xlarge"   | No       |
| amazon_node_min_count       | Specified node count | 1   | No       |
| amazon_node_min_count       | Specified node count | 1   | No       |
| amazon_node_spot_price      | Specified spot proci | 0 (no sport)   | No       |

##
## Test container/plugin with docker

## Build new docker image
    make docker

### Use example .env file and fill require vars
    cp .env.example .env
    docker run -env-file .env --rm -it banzaicloud/pipeline_client:latest




    pipeline:
    ...
        image: banzaicloud/pipeline_client:latest

## Available parameters

### Pipeline api entrypoint (recomended)

    pipeline:
    ...
        image: banzaicloud/pipeline_client:latest
        secrets: [ plugin_endpoint, plugin_username, plugin_password]
        
#### or
    pipeline:
    ...
        image: banzaicloud/pipeline_client:latest
        endpoint: http://[your-host-name-or-ip]/pipeline/api/v1
        username: admin
        password: *****

### Logs (optionals)
    pipeline:
    ...
        log_level: info # optional
        log_format: text # optional
        
## For developers
### Use .env file (example)

    cp .env.example .env
    source .env

### Test with `go run`

    go run -ldflags "-X main.version=1.0" main.go plugin.go log.go --plugin.log.level debug --plugin.log.format text
