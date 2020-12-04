# PlanetScale CLI

### Local Dev Setup
1. Install Go 1.15.5 using [Homebrew](https://brew.sh)

```
brew install golang
```

2. Ensure that your `$GOPATH` is setup properly and that you also have `$GOPATH/bin` added to your `$PATH`.

3. Install the CLI using `go install ./...` or `go install ./cmd/psctl/...` and then execute commands using `psctl <command> <subcommand>`

For logging in, you can just run `psctl auth login`.
