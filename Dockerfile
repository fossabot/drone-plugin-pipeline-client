FROM alpine:3.2
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
ADD bin/linux-amd64/pipeline-client /bin/
ENTRYPOINT ["/bin/pipeline-client"]
