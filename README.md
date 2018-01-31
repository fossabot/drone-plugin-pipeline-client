
## Pipeline API client plugin for Drone

Pipeline REST API client plugin for Drone. A step in the Pipeline PaaS CI/CD component to provision a Kubernetes cluster or use a managed one. Currently two cloud provider supported Amazon and Azure(AKS).

#### Specify required secrets

Provide valid credentials for the pipeline API.

These options needs to be specified in the CI/CD [GUI](https://github.com/banzaicloud/pipeline/blob/master/docs/pipeline-howto.md#cicd-secrets).

* plugin_endpoint: http://[control-plane-host]/pipeline/api/v1
* plugin_username: Specified pipeline username
* plugin_password: Specified pipeline password


### Create or use existing cluster (Amazon EC2)

    pipeline:
      create_cluster:
        cluster_name: "demo-cluster1"
        image: banzaicloud/pipeline_client:latest
        secrets: [ plugin_endpoint, plugin_username, plugin_password ]

### Create or use existing cluster (Azure AKS)
    pipeline:
      create_cluster:
        cluster_name: "demo-cluster1"
        cluster_provider: azure
        image: banzaicloud/pipeline_client:latest
        secrets: [ plugin_endpoint, plugin_username, plugin_password ]

### Main options

| Option           | Description             | Default  | Required |
| -------------    | ----------------------- | --------:| --------:|
| cluster_name     | Specified cluster name  | ""       | Yes      |
| cluster_provider | Specified supporter provider (`amazon`, `azure`) | amazon   | No       |
| log_level        | Specified log level (`info`, `warning`,`error`, `critical`) | info   | No       |
| log_format       | Specified log format (`json`, `text`) | json   | No       |

### Cloud provider specific options

#### Amazon
| Option                      | Description              | Default  | Required |
| -------------               | -----------------------  | --------:| --------:|
| amazon_node_type            | Specified instance type   | "m4.xlarge"       | Yes      |
| amazon_master_image         | Specified image for master node  | ami-06d1667f| No       |
| amazon_master_instance_type | Specified instance type for master node | "m4.xlarge"   | No       |
| amazon_node_image           | Specified image for node | ami-06d1667f| No       |
| amazon_node_instance_type   | Specified instance type for node | "m4.xlarge"   | No       |
| amazon_node_min_count       | Specified node count | 1   | No       |
| amazon_node_min_count       | Specified node count | 1   | No       |
| amazon_node_spot_price      | Specified spot price | 0 (normal instance)   | No       |

#### Azure (AKS)

In case of Azure a resource group has to be used. Use the Azure CLI to create an Azure Resource Group:
https://docs.microsoft.com/en-us/azure/azure-resource-manager/xplat-cli-azure-resource-manager

| Option                      | Description                      | Default  | Required |
| -------------               | -----------------------          | ----------:| --------:|
| azure_resource_group        | Existing azure resource group     | ""        | Yes     |
| azure_kubernetes_version    | Specified kubernetes version | "1.8.2"            | No      |
| azure_agent_name            | Specified agent name         | [cluster_name](#main-options)       | No      |
| azure_node_instance_type    | Specified instance type      | "Standard_D4s_v3"  | No      |

### Dynamic application specific secrets

Applications deployed by CI/CD may require options of which value is unknown until deployment time or doesn't want to specify it directly in `.pipeline.yml` file thus the user will only be able to specify them when hooks the application to the CI/CD flow. Such values can be passed to the application through CI/CD secrets. The values are bound to the keys listed under `deployment_values` -> `app` which is illustrated in the example below.

E.g.:

```yaml
install_my_app:
    image: my_app_docker_image:latest

    deployment_name: "my_app_helm_chart"
    deployment_release_name: "my_app"
    deployment_values:
      app:
        logDirectory: "...."
        db_user: "root"
        db_password: "{{ .PLUGIN_DATABASE_PASSWORD }}"
    secrets: [ plugin_endpoint, plugin_username, plugin_password, plugin_database_password ]
```

In this example beside the [required secrets](#specify-required-secrets) there is a `plugin_database_password` through which we can set up a password through the CI/CD flow. Note the placeholder `{{ .PLUGIN_DATABASE_PASSWORD }}` specified for `plugin_database_password` key in the yaml. This placeholder will be replaced with the value of `plugin_database_password` secret.


Are you a developer? Click [here](dev.md)
