# PlanetScale CLI [![Build status](https://badge.buildkite.com/cf225eb6ccc163b365267fd8172a6e5bd9baa7c8fcdd10c77c.svg?branch=main)](https://buildkite.com/planetscale/cli)

PlanetScale is more than a database and our CLI is more than a jumble of commands. The `pscale` command line tool brings branches, deploy requests, and other PlanetScale concepts to your fingertips.

![PlanetScale CLI](https://user-images.githubusercontent.com/155044/118568235-66c8e380-b745-11eb-8124-5a72e17f7f7b.png)


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

## Documentation

Please checkout our Documentation page: [docs.planetscale.com](https://docs.planetscale.com/reference/planetscale-cli/)
