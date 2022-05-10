FROM golang:1.18 as build
WORKDIR /app
COPY . .

ARG VERSION
ARG COMMIT
ARG DATE

RUN GOOS=linux GOARCH=amd64 go build -ldflags="-X main.commit=$COMMIT -X main.version=$VERSION -X main.date=$DATE" github.com/planetscale/cli/cmd/pscale

FROM alpine:latest  
RUN apk --no-cache add ca-certificates mysql-client
EXPOSE 3306

WORKDIR /app
COPY --from=build /app/pscale /usr/bin
ENTRYPOINT ["/usr/bin/pscale"] 
