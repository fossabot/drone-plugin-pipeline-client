
## Pipeline API client plugin for Drone

Pipeline REST API client plugin for Drone. A step in the Pipeline PaaS CI/CD component to provision a Kubernetes cluster or use a managed one. Currently two cloud provider supported Amazon and Azure(AKS).

#### Specify required secrets

Provide valid credentials for the pipeline API.

These options needs to be specified in the CI/CD [GUI](https://github.com/banzaicloud/pipeline/blob/master/docs/pipeline-howto.md#cicd-secrets).

* endpoint: http://[control-plane-host]/pipeline/api/v1
* username: Specified pipeline username
* password: Specified pipeline password


### Create or use existing cluster (Amazon)

    pipeline:
      create_cluster:
        cluster_name: "demo-cluster1"
        image: banzaicloud/pipeline_client:latest
        secrets: [ endpoint, username, password]

### Create or use existing cluster (Azure)
    pipeline:
      create_cluster:
        cluster_name: "demo-cluster1"
        cluster_provider: azure
        image: banzaicloud/pipeline_client:latest
        secrets: [ endpoint, username, password]

### Main options

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
| amazon_node_spot_price      | Specified spot price | 0 (normal instance)   | No       |

In case of Azure a resource group has to be used. Use the Azure CLI to create Azure Resource Group:
https://docs.microsoft.com/en-us/azure/azure-resource-manager/xplat-cli-azure-resource-manager

#### Azure (AKS)
| Option                      | Description              | Default  | Required |
| -------------               | -----------------------  | --------:| --------:|
| azure_resource_group        | Created azure resource group | ""       | Yes     |
| azure_kubernetes_version    | Desired Kubernetes version   | "1.8.2"  | No      |
| azure_agent_name            | Azure agent name             | ""       | No      |

Are you a developer? Click [here](dev.md)