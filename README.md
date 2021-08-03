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

`pscale` is available as downloadable binary from the [releases](https://github.com/planetscale/cli/releases/latest) page.

#### Manually

Download the pre-compiled binaries from the [releases](https://github.com/planetscale/cli/releases/latest) page and copy to the desired location.


#### Container images 

We provide ready to use Docker container images.  To pull the latest image:

```
docker pull planetscale/pscale:latest
```

To pull a specific version:

```
docker pull planetscale/pscale:v0.62.0
```

## Documentation

Please checkout our Documentation page: [docs.planetscale.com](https://docs.planetscale.com/reference/planetscale-cli/)
