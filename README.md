# PlanetScale CLI [![Build status](https://badge.buildkite.com/cf225eb6ccc163b365267fd8172a6e5bd9baa7c8fcdd10c77c.svg?branch=main)](https://buildkite.com/planetscale/cli)

PlanetScale is more than a database and our CLI is more than a jumble of commands. The `pscale` command line tool brings branches, deploy requests, and other PlanetScale concepts to your fingertips.

![PlanetScale CLI](https://user-images.githubusercontent.com/6104/191803574-be63da54-d255-4f5a-ab2d-2b49cdf7eb12.png)


## Installation

#### macOS

`pscale` is available via a Homebrew Tap, and as downloadable binary from the [releases](https://github.com/planetscale/cli/releases/latest) page:

```
brew install planetscale/tap/pscale
```
Optional: `pscale` requires the MySQL Client for certain commands. You can install it by running:

```
brew install mysql-client
```

To upgrade to the latest version:

```
brew upgrade pscale
```

#### Linux

`pscale` is available as downloadable binaries from the [releases](https://github.com/planetscale/cli/releases/latest) page. Download the .deb or .rpm from the [releases](https://github.com/planetscale/cli/releases/latest) page and install with `sudo dpkg -i` and `sudo rpm -i` respectively.

#### Windows

`pscale` is available via [scoop](https://scoop.sh/), and as a downloadable binary from the [releases](https://github.com/planetscale/cli/releases/latest) page:

```
scoop bucket add pscale https://github.com/planetscale/scoop-bucket.git
scoop install pscale mysql
```

To upgrade to the latest version:

```
scoop update pscale
```

#### Manually

Download the pre-compiled binaries from the [releases](https://github.com/planetscale/cli/releases/latest) page and copy to the desired location.

Alternatively, you can install [bin](https://github.com/marcosnils/bin) which works on all `macOS`, `Windows`, and `Linux` platforms:

```
bin install https://github.com/planetscale/cli
```

To upgrade to the latest version

```
bin upgrade pscale
```

#### Container images 

We provide ready to use Docker container images.  To pull the latest image:

```
docker pull planetscale/pscale:latest
```

To pull a specific version:

```
docker pull planetscale/pscale:v0.63.0
```

If you like to have a shell alias that runs the latest version of pscale from docker whenever you type `pscale`:

```
mkdir -p $HOME/.config/planetscale
alias pscale="docker run -e HOME=/tmp -v $HOME/.config/planetscale:/tmp/.config/planetscale --user $(id -u):$(id -g) --rm -it -p 3306:3306/tcp planetscale/pscale:latest"
```

If you need a more advanced example that works with service tokens and differentiates between commands that need a pseudo terminal or non-interactive mode, [have a look at this shell function](https://github.com/jonico/pscale-cli-helper-scripts/blob/main/.pscale/cli-helper-scripts/use-pscale-docker-image.sh).

## GitHub Actions Usage
Use the [setup-pscale-action](https://github.com/planetscale/setup-pscale-action) to install and use `pscale` in GitHub Actions.

```yaml
- name: Setup pscale
  uses: planetscale/setup-pscale-action@v1
- name: Use pscale
  env:
    PLANETSCALE_SERVICE_TOKEN_ID: ${{ secrets.PLANETSCALE_SERVICE_TOKEN_ID }}
    PLANETSCALE_SERVICE_TOKEN: ${{ secrets.PLANETSCALE_SERVICE_TOKEN }}
  run: |
    pscale deploy-request list my-db --org my-org
```

## Local Development

To run a command:
```
go run cmd/pscale/main.go <command>
```

Alternatively, you can build `pscale`:
```
go build cmd/pscale/main.go
```

And then use the `pscale` binary built in `cmd/pscale/` for testing:
```
./cmd/pscale/pscale <command>
```

## Documentation

Please checkout our Documentation page: [planetscale.com/docs](https://planetscale.com/docs/reference/planetscale-cli)
