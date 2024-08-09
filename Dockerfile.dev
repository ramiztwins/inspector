# This is development dockerfile.
# TODO: generalize this to the production dockerfile (as well as docker-composer.yml)

FROM alpine:latest

RUN apk update && apk add go delve build-base gcc libc6-compat

# A workaround for intellij debugging support
RUN apk update && apk add delve build-base gcc libc6-compat

ENV GOPATH=/go PATH=$GOPATH/bin:/usr/local/go/bin:$PATH

WORKDIR /export

CMD ["go", "run", "main.go", "--config_path", "config.json"]