FROM alpine:3.2
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
ADD pipeline_plugin /bin/
ENTRYPOINT ["/bin/pipeline_plugin"]