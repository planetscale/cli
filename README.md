# PlanetScale CLI

## Installation

**homebrew tap** (only on macOS for now):

```
brew install planetscale/tap/pscale
```
To upgrade to the latest version:

```
brew upgrade pscale
```

**deb/rpm**:

Download the .deb or .rpm from the [releases](https://github.com/planetscale/cli/releases/latest) page and install with dpkg -i and rpm -i respectively.

**manually**:

Download the pre-compiled binaries from the [releases](https://github.com/planetscale/cli/releases/latest) page and copy to the desired location.

### Local Dev Setup

In order to get setup and running with this project, you can run `script/setup` which will install all of the necessary dependencies. If `script/setup` installs Go, you should ensure that your `$GOPATH` is set properly and that you have added `$GOPATH/bin` to your `$PATH`.

You can install and use the program using `script/build` or `go install ./...`. After doing so, you can run the tool using `pscale <command> [subcommand] [flags]`.


For using the CLI locally, there's some extra setup involved to get authenticated.

### Releasing a new version

To release a new version of the CLI make sure to switch to an updated `main` branch:

```
git checkout main
git pull origin main
```

after that create a new tag and push to the repo. Make sure the version is bumped:

```
git tag -a <version> -m <comment>
git push origin <version>
```

This will trigger the CI and invoke `goreleaser`, which will then release all the appropriate packages and archives.


### Updating the vendored API package 


To update the vendored `github.com/planetscale/planetscale-go` API package,
first make sure to set up Go for private repositories:

```
go env -w GOPRIVATE="github.com/planetscale/*"
```

Then, use the following commands to fetch the `latest` HEAD version of the CLI:

```
go get github.com/planetscale/planetscale-go/planetscale
go mod vendor
go mod tidy
```
