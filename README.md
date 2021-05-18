# PlanetScale CLI [![Build status](https://badge.buildkite.com/2307000d934b6a245eb9b84bcac9c1eea1a7c13f010b20c8ba.svg?branch=main)](https://buildkite.com/planetscale/cli)

PlanetScale is more than a database and our CLI is more than a jumble of commands. The `pscale` command line tool brings branches, deploy requests, and other PlanetScale concepts to your finger tips.

![PlanetScale CLI](https://user-images.githubusercontent.com/155044/118568235-66c8e380-b745-11eb-8124-5a72e17f7f7b.png)


## Installation

### macOS

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

### Linux

`pscale` is available as downloadable binaries from the [releases](https://github.com/planetscale/cli/releases/latest) page. Download the .deb or .rpm from the [releases](https://github.com/planetscale/cli/releases/latest) page and install with `sudo dpkg -i` and `sudo rpm -i` respectively.

### Windows

`pscale` is available as downloadable binary from the [releases](https://github.com/planetscale/cli/releases/latest) page.

### Manually

Download the pre-compiled binaries from the [releases](https://github.com/planetscale/cli/releases/latest) page and copy to the desired location.
