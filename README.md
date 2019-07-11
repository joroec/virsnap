# virsnap

virsnap is a CLI snapshot utility for libvirt. The small tool is designed for
creating and automating snapshots of virtual machines (e.g. KVM domains). In
addition to creating snapshots, the tool allows to automatically remove 
deprecated snapshots with ease.

## Example

An example will follow later on.

## Usage

Usage documentation will follow later on.

```shell
$ virsnap -h
```

## Dependencies

virsnap needs go 1.12+.

virsnap uses `vgo` for dependency management. For golang version 1.10 and
lower, `vgo` can be installed as an external tool. As of golang version 1.11,
`vgo` was merged into the golang main tree. For more information on `vgo`, see
the corresponding [vgo documentation].

[vgo documentation]: https://research.swtch.com/vgo-tour

## Installation

To install virsnap, execute the following in your shell:
```shell
git clone http://github.com/joroec/virsnap
cd virsnap
go build
sudo install virsnap /usr/local/bin/virsnap
```

This will compile and link the virsnap binary and install it into
`/usr/local/bin/virsnap`. No other file is installed in system directories.

To remove the tool from your system, execute the following in your shell:
```shell
sudo rm /usr/local/bin/virsnap
```

## Community, discussion, contribution, and support

## FAQ

### How do I install golang 1.12?

You should always prefer your distribution's packet manager for installing
software. If your packet manager does not offer a suitable packet for golang
1.12, you can install it manually. The following are the commands needed for the
manual installation on a Ubuntu 18.04 LTS system.

I prefer to compile software by myself. Alternatively, you can use a release
link to download a binary release of golang. You can find the direct link to
the current go version at the [release page]. The following will build golang
from source:

Visit the official [build guide] for more information.

Newer versions of the golang compiler are written completely in golang.
In compiler construction, there is typically a bootstrap version of a compiler
written in another language (e.g. C) to compile the actual compiler. For golang,
this is the release branch `release-branch.go1.4` which is a bootstrap compiler
for golang written in C. You need to have a GCC or Clang compiler installed
on your system.

Start with downloading the golang source code:

```shell
$ sudo apt-get update && sudo apt-get upgrade
$ git clone https://github.com/golang/go
```

Now, build the bootstrap compiler:
```shell
$ git checkout release-branch.go1.4
$ cd src
$ ./all.bash
...
net/rpc/jsonrpc

Build complete; skipping tests.
To force tests, set GO14TESTS=1 and re-run, but expect some failures.
$ cd ../../
$ cp -r go /tmp/go-bootstrap
```

Now, change back to the desired release brach and compile the golang compiler:
```
cd go
git checkout go1.12.6
git clean -df
cd src
GOROOT_BOOTSTRAP=/tmp/go-bootstrap GOROOT_FINAL=/usr/local/lib/go GOBIN=/usr/local/bin ./make.bash
Building Go cmd/dist using /tmp/go-bootstrap.
Building Go toolchain1 using /tmp/go-bootstrap.
Building Go bootstrap cmd/go (go_bootstrap) using Go toolchain1.
Building Go toolchain2 using go_bootstrap and Go toolchain1.
Building Go toolchain3 using go_bootstrap and Go toolchain2.
Building packages and commands for linux/amd64.
---
Installed Go for linux/amd64 in /home/.../go
Installed commands in /home/.../go/bin

The binaries expect /home/.../go to be copied or moved to /usr/local/lib/go
```

In the last step, you need to install your newly compiled go compiler to your
system:

```shell
cd ../../
sudo cp go /usr/local/lib/go
sudo ln -s /usr/local/lib/go/bin/go /usr/local/bin/go
sudo ln -s /usr/local/lib/go/bin/gofmt /usr/local/bin/gofmt
```

To clean your system from the intermediate products and repositories, you can
execute the following commands:
```
rm -rf /tmp/go-bootstrap
rm -rf go
```

Finally, you can set up your [golang workspace]:
```shell
mkdir ~/.go-workspace
```

and add to your `~/.bash_profile` / `.zshrc`:
```
export GOPATH=$HOME/.go-workspace
```

[build guide]: https://golang.org/doc/install/source
[release page]: https://golang.org/dl/
[golang workspace]: https://golang.org/doc/code.html

## License

MIT License. See `LICENSE` file.

## Authors

See `AUTHORS` file.
