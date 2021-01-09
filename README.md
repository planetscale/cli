# PlanetScale CLI

### Local Dev Setup

In order to get setup and running with this project, you can run `script/setup` which will install all of the necessary dependencies. If `script/setup` installs Go, you should ensure that your `$GOPATH` is set properly and that you have added `$GOPATH/bin` to your `$PATH`.

You can install and use the program using `script/build` or `go install ./...`. After doing so, you can run the tool using `psctl <command> [subcommand] [flags]`.


For using the CLI locally, there's some extra setup involved to get authenticated.

### Updating the vendored API package 


To update the vendored `github.com/planetscale/planetscale-go` API package,
first make sure to set up Go for private repositories:

```
go env -w GOPRIVATE="github.com/planetscale/*"
```

Then, use the following commands to fetch the `latest` HEAD version of the CLI:

```
go get github.com/planetscale/planetscale-go
go mod vendor
go mod tidy
```

#### For logging in: 
1. Create a Rails OAuth application at `http://auth.planetscaledb.local:3000/oauth/applications/new`. Make sure the `Confidential` box is unchecked and that the scopes are `read_database write_database`. You can make the `Redirect URI` be anything.
2. After creating this application, keep note of the client ID.
3. When logging in, you can override the default client ID by using the `--client-id [uid]` flag to override the default client ID with your local one. You can override the API URL to be `http://auth.planetscaledb.local:3000` (or whatever your Rails server is) by passing in the `--api-url http://auth.planetscaledb.local:3000/` flag. An example of the login command will look like `psctl auth login --client-id your_uid --api-url http://auth.planetscaledb.local:3000/`.


#### For testing out other endpoints:

You can use the following structure for testing locally: `psctl db create --api-url http://api.planetscaledb.local:3000/`. This will always use the local API server for testing.
