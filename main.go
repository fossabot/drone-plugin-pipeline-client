package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/template"

	. "github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/banzai-types/components/amazon"
	"github.com/banzaicloud/banzai-types/components/azure"
	"github.com/banzaicloud/banzai-types/components/google"
	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var (
	version                string = ""
	defaultAmazonImage     string = "ami-06d1667f"
	defaultAmazonSpotPrice string = "0.2" //spot price for the default region/instance type

	defaultInstanceType = map[string]string{
		"amazon": "m4.xlarge",       // 4 vCPU, 16 GB RAM, General Purpose
		"azure":  "Standard_D4s_v3", // 4 vCPU, 16 GB RAM, General Purpose
		"google": "n1-standard-4",   // 4 vCPUs 15 GB RAM. Standard machine
	}
	defaultClusterLocation = map[string]string{
		"amazon": "eu-west-1",
		"azure":  "westeurope",
		"google": "us-central1-a",
	}
)

func main() {

	_ = godotenv.Load()

	app := cli.NewApp()
	app.Name = "Pipeline client"
	app.Usage = "Pipeline client"
	app.Action = run
	app.Version = fmt.Sprintf("%s", version)
	app.EnableBashCompletion = true
	app.Flags = []cli.Flag{

		//
		// repo args
		//

		cli.StringFlag{
			Name:   "repo.fullname",
			Usage:  "repository full name",
			EnvVar: "DRONE_REPO",
		},
		cli.StringFlag{
			Name:   "repo.owner",
			Usage:  "repository owner",
			EnvVar: "DRONE_REPO_OWNER",
		},
		cli.StringFlag{
			Name:   "repo.name",
			Usage:  "repository name",
			EnvVar: "DRONE_REPO_NAME",
		},
		cli.StringFlag{
			Name:   "repo.link",
			Usage:  "repository link",
			EnvVar: "DRONE_REPO_LINK",
		},
		cli.StringFlag{
			Name:   "repo.avatar",
			Usage:  "repository avatar",
			EnvVar: "DRONE_REPO_AVATAR",
		},
		cli.StringFlag{
			Name:   "repo.branch",
			Usage:  "repository default branch",
			EnvVar: "DRONE_REPO_BRANCH",
		},
		cli.BoolFlag{
			Name:   "repo.private",
			Usage:  "repository is private",
			EnvVar: "DRONE_REPO_PRIVATE",
		},
		cli.BoolFlag{
			Name:   "repo.trusted",
			Usage:  "repository is trusted",
			EnvVar: "DRONE_REPO_TRUSTED",
		},

		//
		// commit args
		//

		cli.StringFlag{
			Name:   "remote.url",
			Usage:  "git remote url",
			EnvVar: "DRONE_REMOTE_URL",
		},
		cli.StringFlag{
			Name:   "build.path",
			Usage:  "workspace dir",
			EnvVar: "PLUGIN_PATH,DRONE_WORKSPACE",
		},
		cli.StringFlag{
			Name:   "commit.sha",
			Usage:  "git commit sha",
			EnvVar: "DRONE_COMMIT_SHA",
		},
		cli.StringFlag{
			Name:   "commit.ref",
			Value:  "refs/heads/master",
			Usage:  "git commit ref",
			EnvVar: "DRONE_COMMIT_REF",
		},
		cli.StringFlag{
			Name:   "commit.branch",
			Value:  "master",
			Usage:  "git commit branch",
			EnvVar: "DRONE_COMMIT_BRANCH",
		},
		cli.StringFlag{
			Name:   "commit.message",
			Usage:  "git commit message",
			EnvVar: "DRONE_COMMIT_MESSAGE",
		},
		cli.StringFlag{
			Name:   "commit.link",
			Usage:  "git commit link",
			EnvVar: "DRONE_COMMIT_LINK",
		},
		cli.StringFlag{
			Name:   "commit.author.name",
			Usage:  "git author name",
			EnvVar: "DRONE_COMMIT_AUTHOR",
		},
		cli.StringFlag{
			Name:   "commit.author.email",
			Usage:  "git author email",
			EnvVar: "DRONE_COMMIT_AUTHOR_EMAIL",
		},
		cli.StringFlag{
			Name:   "commit.author.avatar",
			Usage:  "git author avatar",
			EnvVar: "DRONE_COMMIT_AUTHOR_AVATAR",
		},

		//
		// build args
		//

		cli.StringFlag{
			Name:   "build.event",
			Value:  "push",
			Usage:  "build event",
			EnvVar: "DRONE_BUILD_EVENT",
		},
		cli.IntFlag{
			Name:   "build.number",
			Usage:  "build number",
			EnvVar: "DRONE_BUILD_NUMBER",
		},
		cli.IntFlag{
			Name:   "build.created",
			Usage:  "build created",
			EnvVar: "DRONE_BUILD_CREATED",
		},
		cli.IntFlag{
			Name:   "build.started",
			Usage:  "build started",
			EnvVar: "DRONE_BUILD_STARTED",
		},
		cli.IntFlag{
			Name:   "build.finished",
			Usage:  "build finished",
			EnvVar: "DRONE_BUILD_FINISHED",
		},
		cli.StringFlag{
			Name:   "build.status",
			Usage:  "build status",
			Value:  "success",
			EnvVar: "DRONE_BUILD_STATUS",
		},
		cli.StringFlag{
			Name:   "build.link",
			Usage:  "build link",
			EnvVar: "DRONE_BUILD_LINK",
		},
		cli.StringFlag{
			Name:   "build.deploy",
			Usage:  "build deployment target",
			EnvVar: "DRONE_DEPLOY_TO",
		},
		cli.BoolFlag{
			Name:   "yaml.verified",
			Usage:  "build yaml is verified",
			EnvVar: "DRONE_YAML_VERIFIED",
		},
		cli.BoolFlag{
			Name:   "yaml.signed",
			Usage:  "build yaml is signed",
			EnvVar: "DRONE_YAML_SIGNED",
		},

		//
		// prev build args
		//

		cli.IntFlag{
			Name:   "prev.build.number",
			Usage:  "previous build number",
			EnvVar: "DRONE_PREV_BUILD_NUMBER",
		},
		cli.StringFlag{
			Name:   "prev.build.status",
			Usage:  "previous build status",
			EnvVar: "DRONE_PREV_BUILD_STATUS",
		},
		cli.StringFlag{
			Name:   "prev.commit.sha",
			Usage:  "previous build sha",
			EnvVar: "DRONE_PREV_COMMIT_SHA",
		},

		//
		// plugin args
		//

		cli.StringFlag{
			Name:   "plugin.endpoint",
			Usage:  "API Url",
			EnvVar: "PLUGIN_ENDPOINT,ENDPOINT",
		},
		cli.StringFlag{
			Name:   "plugin.username",
			Usage:  "API Username",
			EnvVar: "PLUGIN_USERNAME,USERNAME",
		},
		cli.StringFlag{
			Name:   "plugin.password",
			Usage:  "API Password",
			EnvVar: "PLUGIN_PASSWORD,PASSWORD",
		},
		cli.StringFlag{
			Name:   "plugin.token",
			Usage:  "API OAuth Token",
			EnvVar: "PLUGIN_TOKEN,TOKEN",
		},
		cli.StringFlag{
			Name:   "plugin.cluster.name",
			Usage:  "K8S Cluster name",
			EnvVar: "PLUGIN_CLUSTER_NAME",
		},
		cli.StringFlag{
			Name:   "plugin.cluster.location",
			Usage:  "Specific aws region",
			EnvVar: "PLUGIN_CLUSTER_LOCATION",
		},
		cli.StringFlag{
			Name:   "plugin.cluster.state",
			Usage:  "K8S cluster desire state",
			EnvVar: "PLUGIN_CLUSTER_STATE",
			Value:  "created",
		},
		cli.StringFlag{
			Name:   "plugin.cluster.provider",
			Usage:  "K8S cluster provider",
			EnvVar: "PLUGIN_CLUSTER_PROVIDER",
			Value:  "amazon",
		},
		cli.StringFlag{
			Name:   "plugin.node.instance_type",
			Usage:  "EC2 instance type",
			EnvVar: "PLUGIN_AMAZON_NODE_INSTANCE_TYPE,PLUGIN_AZURE_NODE_INSTANCE_TYPE, PLUGIN_GOOGLE_INSTANCE_TYPE",
		},
		cli.StringFlag{
			Name:   "plugin.azure.resource_group",
			Usage:  "Azure resource group name",
			EnvVar: "PLUGIN_AZURE_RESOURCE_GROUP",
		},
		cli.StringFlag{
			Name:   "plugin.azure.node.count",
			Usage:  "Azure resource group name",
			EnvVar: "PLUGIN_AZURE_NODE_COUNT",
		},
		cli.StringFlag{
			Name:   "plugin.azure.kubernetes_version",
			Usage:  "Azure kubernetes version",
			EnvVar: "PLUGIN_AZURE_KUBERNETES_VERSION",
			Value:  "1.8.2",
		},
		cli.StringFlag{
			Name:   "plugin.azure.agent_name",
			Usage:  "Azure agent name",
			EnvVar: "PLUGIN_AZURE_AGENT_NAME",
		},
		cli.StringFlag{
			Name:   "plugin.amazon.node.image",
			Usage:  "Amazon machine image id",
			EnvVar: "PLUGIN_AMAZON_NODE_IMAGE",
			Value:  defaultAmazonImage,
		},
		cli.StringFlag{
			Name:   "plugin.amazon.node.instance_type",
			Usage:  "Amazon instance type",
			EnvVar: "PLUGIN_AMAZON_NODE_INSTANCE_TYPE",
		},
		cli.IntFlag{
			Name:   "plugin.amazon.node.min_count",
			Usage:  "Minimum number of node",
			EnvVar: "PLUGIN_AMAZON_NODE_MIN_COUNT",
			Value:  1,
		},
		cli.IntFlag{
			Name:   "plugin.amazon.node.max_count",
			Usage:  "Maximum number of node",
			EnvVar: "PLUGIN_AMAZON_NODE_MAX_COUNT",
			Value:  1,
		},
		cli.StringFlag{
			Name:   "plugin.amazon.node.spot_price",
			Usage:  "Spot price",
			EnvVar: "PLUGIN_AMAZON_NODE_SPOT_PRICE",
			Value:  defaultAmazonSpotPrice,
		},
		cli.StringFlag{
			Name:   "plugin.amazon.master.image",
			Usage:  "Amazon machine image id",
			EnvVar: "PLUGIN_AMAZON_MASTER_IMAGE",
		},
		cli.StringFlag{
			Name:   "plugin.amazon.master.instance_type",
			Usage:  "EC2 instance type",
			EnvVar: "PLUGIN_AMAZON_MASTER_INSTANCE_TYPE",
		},
		cli.StringFlag{
			Name:   "plugin.deployment.name",
			Usage:  "Specific deployment name",
			EnvVar: "PLUGIN_DEPLOYMENT_NAME",
		},
		cli.StringFlag{
			Name:   "plugin.deployment.release_name",
			Usage:  "Specific deployment release name",
			EnvVar: "PLUGIN_DEPLOYMENT_RELEASE_NAME",
			Value:  "default",
		},
		cli.StringFlag{
			Name:   "plugin.deployment.state",
			Usage:  "Specific deployment state",
			EnvVar: "PLUGIN_DEPLOYMENT_STATE",
			Value:  "created",
		},
		cli.StringFlag{
			Name:   "plugin.deployment.values",
			Usage:  "Specific deployment values",
			EnvVar: "PLUGIN_DEPLOYMENT_VALUES",
		},
		cli.StringFlag{
			Name:   "plugin.log.level",
			Usage:  "Specific log level (debug,info,warn)",
			EnvVar: "PLUGIN_LOG_LEVEL",
			Value:  "info",
		},
		cli.StringFlag{
			Name:   "plugin.log.format",
			Usage:  "Specific log format (text, json) default is text",
			EnvVar: "PLUGIN_LOG_FORMAT",
			Value:  "text",
		},
		cli.StringFlag{
			Name:   "plugin.google.project",
			Usage:  "The google cloud project name",
			EnvVar: "PLUGIN_GOOGLE_PROJECT",
		},
		cli.StringFlag{
			Name:   "plugin.google.gke.version",
			Usage:  "The kubernetes version of the GKE",
			EnvVar: "PLUGIN_GOOGLE_GKE_VERSION",
			Value:  "1.8.7-gke.0",
		},
		cli.IntFlag{
			Name:   "plugin.google.node.count",
			Usage:  "The number of nodes in the cluster",
			EnvVar: "PLUGIN_GOOGLE_NODE_COUNT",
			Value:  1,
		},
		cli.StringFlag{
			Name:   "plugin.google.service.account",
			Usage:  "The service account the  cluster instance are run as",
			EnvVar: "PLUGIN_GOOGLE_SERVICE_ACCOUNT",
		},
	}
	app.Run(os.Args)
}

func settingUpDefaults(c *cli.Context) {

	provider := c.String("plugin.cluster.provider")

	if c.String("plugin.node.instance_type") == "" {
		c.Set("plugin.node.instance_type", defaultInstanceType[provider])
	}

	if c.String("plugin.amazon.master.instance_type") == "" {
		c.Set("plugin.amazon.master.instance_type", defaultInstanceType[provider])
	}

	if c.String("plugin.cluster.location") == "" {
		c.Set("plugin.cluster.location", defaultClusterLocation[provider])
	}

	if c.String("plugin.azure.agent_name") == "" {
		c.Set("plugin.azure.agent_name", c.String("plugin.cluster.name"))
	}

}

func run(c *cli.Context) error {

	excludeVars := map[string]bool{
		"PLUGIN_ENDPOINT": true,
		"ENDPOINT":        true,
		"PLUGIN_USERNAME": true,
		"USERNAME":        true,
		"PLUGIN_PASSWORD": true,
		"PASSWORD":        true,
	}

	items := map[string]string{}
	itemForTemplate := map[string]string{}
	for _, element := range os.Environ() {
		variable := strings.SplitN(element, "=", 2)

		if strings.Contains(variable[0], "PLUGIN") && !excludeVars[variable[0]] {
			items[variable[0]] = variable[1]
		}

		if !strings.Contains(variable[0], "PLUGIN") && !excludeVars[variable[0]] {
			itemForTemplate[variable[0]] = variable[1]
		}
	}

	if _, err := os.Stat(".env"); os.IsNotExist(err) {
		_ = godotenv.Write(items, ".env")
	}

	processLogLevel(c)
	settingUpDefaults(c)

	var deploymentValues map[string]interface{}
	var deploymentValStr = c.String("plugin.deployment.values")

	if deploymentValStr != "" {

		err := json.Unmarshal([]byte(processDeploymentSecrets(deploymentValStr, itemForTemplate)), &deploymentValues)

		log.Debugf("deployment values %#v", deploymentValues)

		if err != nil {
			log.Fatalf("unable to parse deployment values: %+v", err)
		}
	}

	plugin := Plugin{
		Repo: Repo{
			Owner:   c.String("repo.owner"),
			Name:    c.String("repo.name"),
			Link:    c.String("repo.link"),
			Avatar:  c.String("repo.avatar"),
			Branch:  c.String("repo.branch"),
			Private: c.Bool("repo.private"),
			Trusted: c.Bool("repo.trusted"),
		},
		Build: Build{
			Number:   c.Int("build.number"),
			Event:    c.String("build.event"),
			Status:   c.String("build.status"),
			Deploy:   c.String("build.deploy"),
			Created:  int64(c.Int("build.created")),
			Started:  int64(c.Int("build.started")),
			Finished: int64(c.Int("build.finished")),
			Path:     c.String("build.path"),
			Link:     c.String("build.link"),
		},
		Commit: Commit{
			Remote:  c.String("remote.url"),
			Sha:     c.String("commit.sha"),
			Ref:     c.String("commit.sha"),
			Link:    c.String("commit.link"),
			Branch:  c.String("commit.branch"),
			Message: c.String("commit.message"),
			Author: Author{
				Name:   c.String("commit.author.name"),
				Email:  c.String("commit.author.email"),
				Avatar: c.String("commit.author.avatar"),
			},
		},
		Config: Config{
			Endpoint: c.String("plugin.endpoint"),
			Username: c.String("plugin.username"),
			Password: c.String("plugin.password"),
			Token:    c.String("plugin.token"),

			Cluster: &CustomCluster{
				CreateClusterRequest: &CreateClusterRequest{
					Name:             c.String("plugin.cluster.name"),
					Location:         c.String("plugin.cluster.location"),
					Cloud:            c.String("plugin.cluster.provider"),
					NodeInstanceType: c.String("plugin.node.instance_type"),
					Properties: struct {
						CreateClusterAmazon *amazon.CreateClusterAmazon `json:"amazon,omitempty"`
						CreateClusterAzure  *azure.CreateClusterAzure   `json:"azure,omitempty"`
						CreateClusterGoogle *google.CreateClusterGoogle `json:"google,omitempty"`
					}{
						CreateClusterAmazon: &amazon.CreateClusterAmazon{
							Node: &amazon.CreateAmazonNode{
								SpotPrice: c.String("plugin.amazon.node.spot_price"),
								MinCount:  c.Int("plugin.amazon.node.min_count"),
								MaxCount:  c.Int("plugin.amazon.node.max_count"),
								Image:     c.String("plugin.amazon.node.image"),
							},
							Master: &amazon.CreateAmazonMaster{
								InstanceType: c.String("plugin.amazon.master.instance_type"),
								Image:        c.String("plugin.amazon.master.image"),
							},
						},
						CreateClusterAzure: &azure.CreateClusterAzure{
							Node: &azure.CreateAzureNode{
								ResourceGroup:     c.String("plugin.azure.resource_group"),
								AgentCount:        c.Int("plugin.azure.node.count"),
								AgentName:         c.String("plugin.azure.agent_name"),
								KubernetesVersion: c.String("plugin.azure.kubernetes_version"),
							},
						},
						CreateClusterGoogle: &google.CreateClusterGoogle{
							Project: c.String("plugin.google.project"),
							Master: &google.GoogleMaster{
								Version: c.String("plugin.google.gke.version"),
							},
							Node: &google.GoogleNode{
								Version: c.String("plugin.google.gke.version"),
								Count:   c.Int("plugin.google.node.count"),
							},
						},
					},
				},
				State: c.String("plugin.cluster.state"),
			},
			Deployment: &Deployment{
				Name:        c.String("plugin.deployment.name"),
				ReleaseName: c.String("plugin.deployment.release_name"),
				State:       c.String("plugin.deployment.state"),
				Values:      deploymentValues,
			},
		},
	}

	plugin.processServiceAccount(c)

	err := plugin.Exec()
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

// Replaces placeholders in the deployment values Go template
// Fails if invalid template provided as deployment value.
func processDeploymentSecrets(deploymentValuesStr string, pluginEnv map[string]string) string {
	log.Infof("filling secrets in deployment values...")

	deplValTpl, err := template.New("depValTpl").Parse(deploymentValuesStr)
	if err != nil {
		log.Fatalf("%#err", errors.New(fmt.Sprintf("failed to create template: [%s]", err.Error())))
	}

	var tpl bytes.Buffer
	err = deplValTpl.ExecuteTemplate(&tpl, "depValTpl", pluginEnv)
	if err != nil {
		log.Fatalf("%#err", errors.New(fmt.Sprintf("failed to execute template: [%s]", err.Error())))
	}
	log.Info("secrets filled in deployment values.")
	return tpl.String()
}

func (plugin *Plugin) processServiceAccount(c *cli.Context) {
	serviceAccount := c.String("plugin.google.service.account")

	if len(serviceAccount) > 0 {
		plugin.Config.Cluster.Properties.CreateClusterGoogle.Node.ServiceAccount = serviceAccount
	}
}

func processLogLevel(c *cli.Context) {
	switch strings.ToUpper(c.String("plugin.log.level")) {
	case "INFO":
		log.SetLevel(log.InfoLevel)
	case "DEBUG":
		log.SetLevel(log.DebugLevel)
	case "WARN":
		log.SetLevel(log.WarnLevel)
	case "ERROR":
		log.SetLevel(log.ErrorLevel)
	case "PANIC":
		log.SetLevel(log.PanicLevel)
	}
}
