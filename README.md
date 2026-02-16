# health_checker

## Objective
Create an automated health check system for a web application that verifies the application's availability and performance. The system should be capable of sending alerts when certain thresholds are exceeded, suggesting potential issues with the service.

## Prerequisites

All of the tools below were tested in this project on latest versions. I don't think there should be any issues with using older versions but I have not tested it :D

- golang 
- helm
- kind
- kubectl
- opentofu/terraform
- docker

If you would like to try out Nix and devenv, I have included some instructions for that below, but it is not required to run the project. The main idea behind using Nix, devenv, and direnv is to define the packages needed to work with a repository in a declarative way, and then have the development environment automatically set up when you change into the directory. Ask me a bit about it if you are interested, I think it is a really nice way to manage development environments and dependencies :D

### Nix

I get another chance to proliferate Nix, which I think is awesome as a package manager and bundling this with devenv adds a really slim and easy to use abstraction on top of it that I think brings a lot of value :D 

#### Option A: Determinate Systems Installer

To install nix, I usually follow the [Determinate System installation](https://github.com/DeterminateSystems/nix-installer?tab=readme-ov-file#install-determinate-nix) because they have a cleaner way of rolling back failed installations. 

``` bash
curl --proto '=https' --tlsv1.2 -sSf -L https://install.determinate.systems/nix | sh -s -- install
```

1. Select no on the first prompt which is to use the Determinate Systems Nix package instead of upstream one. This has caused issues for me in the past if using their fork of nixpkgs.
2. Follow the rest of the prompts, it is safe to accept the defaults afterwards.

``` bash
nix --version
# Should show something like: nix (Nix) 2.33.3
```

#### Option B: Offocial Nix Installer

Visit https://nixos.org/download and follow the installation instructions for your operating system. I used to not recommend this but they claim to have fixed some of the MacOS issues with installation so it may be worth a try.


### Devenv + Direnv

``` bash
nix profile add nixpkgs#direnv --extra-experimental-features nix-command --extra-experimental-features flakes
nix profile add  nixpkgs#devenv --extra-experimental-features nix-command --extra-experimental-features flakes --accept-flake-config
```

To allow direnv to automatically load the environment when changing into the directory, you need to update your shell configuration: [instructions](https://direnv.net/docs/hook.html)

The first time you change into the directory, it will ask you to allow the `devenv.nix` file, this is a security feature of direnv to prevent automatically executing code when changing into a directory. You can review the file and then allow it by running `direnv allow` in the terminal. After this, every time you change into the repository directory, the environment will be automatically set up with the packages defined in `devenv.nix` and any scripts defined there will be available to run as well, with autocompletion!


## Running the project

### Docker Compose

The simplest way to get things running is to run:

``` bash
docker compose up -d
```

This will start the health-check service, a local elasticsearch instance, prometheus, and grafana. You can then access a simple grafana dashboard at `http://localhost:3000/d/b91bae41-656b-409a-8d99-45609523d5cd`.


### K8s

Some of the following scripts are defined in the `devenv.nix` file as well which one can run the commands defined there.

1. Set up the Kind cluster:

``` bash
cluster-up
```

2. Build the docker image and load it into the kind cluster:

``` bash
load-docker-image
```

3. Deploy the helm chart:

a. Using helm directly:
``` bash
helm-deploy
```

b. Using tofu/terraform: 
``` bash
cd tofu/
tofu apply -auto-approve
```