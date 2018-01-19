## Pipeline API client plugin for Drone (Developer Guide)
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