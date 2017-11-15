FROM alpine:3.2
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
ADD pipeline-client /bin/
ENTRYPOINT ["/bin/pipeline-client"]
