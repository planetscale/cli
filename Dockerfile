FROM golang:1.16 as build
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build github.com/planetscale/cli/cmd/pscale

FROM alpine:latest  
RUN apk --no-cache add ca-certificates mysql-client

WORKDIR /app
COPY --from=build /app/pscale /usr/bin
ENTRYPOINT ["/usr/bin/pscale"] 
