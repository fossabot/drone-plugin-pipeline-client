package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"github.com/urfave/cli"
	"os"
)

var (
	version string = ""
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
			EnvVar: "PLUGIN_ENDPOINT",
		},
		cli.StringFlag{
			Name:   "plugin.username",
			Usage:  "API Username",
			EnvVar: "PLUGIN_USERNAME",
		},
		cli.StringFlag{
			Name:   "plugin.password",
			Usage:  "API Password",
			EnvVar: "PLUGIN_PASSWORD",
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
			Value:  "present",
		},
		cli.StringFlag{
			Name:   "plugin.node.image",
			Usage:  "Amazon machine image id",
			EnvVar: "PLUGIN_NODE_IMAGE",
		},
		cli.StringFlag{
			Name:   "plugin.node.instanceType",
			Usage:  "EC2 instance type",
			EnvVar: "PLUGIN_NODE_INSTANCE_TYPE",
		},
		cli.IntFlag{
			Name:   "plugin.node.minCount",
			Usage:  "Minimum number of node",
			EnvVar: "PLUGIN_NODE_MIN_COUNT",
			Value:  1,
		},
		cli.IntFlag{
			Name:   "plugin.node.maxCount",
			Usage:  "Maximum number of node",
			EnvVar: "PLUGIN_NODE_MAX_COUNT",
			Value:  1,
		},
		cli.Float64Flag{
			Name:   "plugin.node.spotPrice",
			Usage:  "Spot price",
			EnvVar: "PLUGIN_NODE_SPOT_PRICE",
			Value:  0,
		},
		cli.StringFlag{
			Name:   "plugin.master.image",
			Usage:  "Amazon machine image id",
			EnvVar: "PLUGIN_MASTER_IMAGE",
		},
		cli.StringFlag{
			Name:   "plugin.master.instanceType",
			Usage:  "EC2 instance type",
			EnvVar: "PLUGIN_MASTER_INSTANCE_TYPE",
		},
		cli.StringFlag{
			Name:   "plugin.deployment.name",
			Usage:  "Specific deployment name",
			EnvVar: "PLUGIN_DEPLOYMENT_NAME",
		},
		cli.StringFlag{
			Name:   "plugin.deployment.state",
			Usage:  "Specific deployment state",
			EnvVar: "PLUGIN_DEPLOYMENT_STATE",
		},

		cli.StringFlag{
			Name:   "plugin.log.level",
			Usage:  "Specific log level (debug,info,warn)",
			EnvVar: "PLUGIN_LOG_LEVEL",
		},
		cli.StringFlag{
			Name:   "plugin.log.format",
			Usage:  "Specific log format (text, json) default is text",
			EnvVar: "PLUGIN_LOG_FORMAT",
			Value:  "text",
		},
	}
	app.Run(os.Args)
}

func run(c *cli.Context) error {

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
			Cluster: Cluster{
				Name:     c.String("plugin.cluster.name"),
				Location: c.String("plugin.cluster.location"),
				State:    c.String("plugin.cluster.state"),
				Node: Node{
					Image:        c.String("plugin.node.image"),
					InstanceType: c.String("plugin.node.instanceType"),
					MinCount:     c.Int("plugin.node.minCount"),
					MaxCount:     c.Int("plugin.node.maxCount"),
					SpotPrice:    c.Float64("plugin.node.spotPrice"),
				},
				Master: Master{
					Image:        c.String("plugin.master.image"),
					InstanceType: c.String("plugin.master.instanceType"),
				},
				Deployment: Deployment{
					Name:  c.String("plugin.deployment.name"),
					State: c.String("plugin.deployment.state"),
				},
			},
		},
	}

	SetLogLevel(c.String("plugin.log.level"))
	SetLogFormat(c.String("plugin.log.format"))

	err := plugin.Exec()
	if err != nil {
		Fatal(err)
	}
	return nil
}
