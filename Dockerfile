FROM golang:alpine
LABEL maintainer="Marcel Juhnke <marcel.juhnke@karrieretutor.de>"

ENV GOPATH /go

RUN apk add --update git

WORKDIR /go

COPY . src/github.com/karrieretutor/b2c-group-membership

RUN go get -v github.com/karrieretutor/b2c-group-membership

FROM alpine:latest
RUN apk --no-cache add ca-certificates

COPY --from=0 /go/bin/b2c-group-membership /usr/local/bin

# Expose the ports we need and setup the ENTRYPOINT w/ the default argument
# to be pass in.

CMD [ "b2c-group-membership" ]