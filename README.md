# route

The routing service handles passing requests to the provider endpoints.

## Status

The current implementation draws on a lot of work done by the GOV.UK team in their router.
They [wrote](https://gdstechnology.blog.gov.uk/2013/12/05/building-a-new-router-for-gov-uk/) about
some of their experiences and shared their progress, much of which is the basis for this. The muxer has been
included and modified, however, to allow for logging.

Route definitions are defined in a simple way in the `hosts.conf` file.

## Installation on Ubuntu

The following instructions will let you run the router component on a fresh, Ubuntu 12.04 x64 machine or VM. The instructions have been verified on the `precise64` Vagrant box. As expected, more information is available in the Go [install docs](http://golang.org/doc/install).

1. Visit the [downloads list](https://code.google.com/p/go/downloads/list). For our machine, we will be choosing the 1.2.1 Linux amd64 binary build.
4. `wget https://go.googlecode.com/files/go1.2.1.linux-amd64.tar.gz`
	- This will download the appropriate binary to your working directory.
5. `sudo tar -C /usr/local -xzf go1.2.1.linux-amd64.tar.gz`
	- This will extract the binary and supporting files to your `/usr/local` path.
6. `echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee -a /etc/profile`
	- This will add a **global** entry to your `/etc/profile`, which is the global equivalent of your `.bashrc`.
	- You may also choose to run `echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc` to simply add the entry to your `.bashrc`.
7. Now, close and reopen your terminal of choice in order to update your path. Alternatives include sourcing your just-modified Bash profile and manually manipulating your path with `export`.
8. `go version`
	- Now, you should be able to check the installed Go version.
9. `mkdir ~/workspace`
	- We are moving on to setting up a Go workspace to install our router into.
	- Note that Go supports a per-user workspace directory. This is referred to as `$GOPATH` and is changed by manipulating that environment variable.
	- Go does not support arbitrarily sprinkling installations throughout your filesystem. This allows Go to manage dependency installations in one place. Savvy developers may anticipate interesting complexities with versioning dependencies.
	- The takeway here is **you only set your `$GOPATH` once** and that is considered your workspace for **all Go projects**.
10. `cd ~/workspace`
	- Move into your new Go workspace. Roomy!
11. `export GOPATH=$HOME/workspace`
	- Set the `$GOPATH` environment variable to be the new workspace you just moved into.
12. `sudo apt-get install git mercurial`
	- In preparation for installing dependencies, we will need the `git` and `hg` commands available. Go tries to support major the VCS systems and many developers do use Mercurial.
13. `go get github.umn.edu/umnapi/route.git`
	- In one fell swoop, clone our router repository, download the required dependencies, build any dependencies as well as our project, and place the binaries in `~/workspace/bin`.
	- You will have to authenticate against enterprise GitHub for each project hosted there. In other words, without SSH keys, this may involve multiple credential entries.
14. `bin/route.git`
	- Start the router.
	- The `.git` extension on this binary is due to enterprise GitHub. In fact, public GitHub is [a special case](https://groups.google.com/forum/#!topic/golang-nuts/AURCoVLjNyc), which is why `.git` extensions are not present for repositories cloned from there.
	- This binary is independent. You can pick it up now and move it somewhere else to run it. You can copy it somewhere and rename it. It includes everything necessary to run itself.
