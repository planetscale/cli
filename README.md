# PlanetScale CLI

### Local Dev Setup
1. Install Go 1.15.5 using [Homebrew](https://brew.sh)

```
brew install golang
```

2. Ensure that your `$GOPATH` is setup properly and that you also have `$GOPATH/bin` added to your `$PATH`.

3. Install the CLI using `go install ./...` or `go install ./cmd/psctl/...` and then execute commands using `psctl <command> <subcommand>`


For testing locally, you will need to do the following steps.

For logging in: 
1. Create a Rails OAuth application at `/http://localhost:3000/oauth/applications/new`. Make sure the `Confidential` box is unchecked and that the scopes are `read_database write_database`. You can make the `Redirect URI` be anything.
2. After creating this application, keep note of the client ID.
3. When logging in, you can override the default client ID by using the `--client-id [uid]` flag to override the default client ID with your local one. You can override the API URL to be `http://localhost:3000` (or whatever your Rails server is) by passing in the `--api-url http://localhost:3000/` flag. An example of the login command will look like `psctl auth login --client-id your_uid --api-url http://localhost:3000/`.


For testing out other endpoints:

You can use the following structure for testing locally: `psctl db create --api-url http://localhost:3000/`. This will always use the local API server for testing.
