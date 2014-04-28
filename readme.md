# route

The routing service handles authenticating and passing requests to the provider endpoints.

## Status

Currnetly, the router is able to authenticate requests according to our spec, and then forward them to endpoints defined in the configuration. It also has a very minimal administrative API. It interfaces with the `store` project as well and caches data to Redis.

## Usage

Testing and usage is available via the Vagrant setup included here. By running `vagrant up` from the project directory, you will have a box with Go, ZeroMQ and Redis available for you. It also pre-installs more complex dependencies. For the curious, the provisioning is handled provincially through the `setup.sh` script. The script is not wasteful on multiple runs.

Note that there may be compile errors during the provisioning process; this is fine.

After setup, two more commands are required.

1. `cd ~/workspace && go get api.umn.edu/route`
	- In one fell swoop, clone our router repository, download the required dependencies, build any dependencies as well as our project, and place the binaries in `~/workspace/bin`.
	- You will have to authenticate against enterprise GitHub for each project hosted there. In other words, without SSH keys, this may involve multiple credential entries.
2. `bin/route`
	- Start the router.
	- Note that without the storage server running, there will be an error message.
