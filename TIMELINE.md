# Project Timeline

The idea behind this timeline is to track some of the things I started to investigate and learn about so that the team gets a better understanding of where I am, and how I think. This should also help clarify some of the technical decisions I made, and why I made them. I understand in the age of LLMs, this might be a good idea as proof of individual work as well :D

## Init

I have had a lot of success with managing packages with Nix+devenv+direnv, specifically devenv has been a nice abstraction over Nix to make it easy to unify the development environment across teams for each repository (you get everything you need to contribute by just changing directory). I plan to also hook in some QoL scripts and pre-commit hooks into the env for the dev loop.

## Go Structure

I don't have a lot of production experience developing a large project in Go, so I am a little bit uncertain what the appropriate ways to structure projects are. This will be small, so I will keep it simple for now, but I did some reading: https://go.dev/doc/modules/layout, https://github.com/golang-standards/project-layout (this seemed like overkill).

## Setting up vanilla server with configs

There's not much more to structure for now, I want to start implementing some of the requirements and see where I get. Again I will keep it simple, start with envvars because then I don't have to worry about unmarshelling yaml configs for now, maybe I will loop back and allow both config and env vars, but env vars are simpler to start with.

## Start calling out to the services

Now that we have a simple server in place that will be used for serving the prom metrics later (note for later: https://prometheus.io/docs/guides/go-application/), we need to start the timed loop of reaching out to the configured endpoint.

Find https://stackoverflow.com/questions/16466320/is-there-a-way-to-do-repetitive-tasks-at-intervals, points to using time.NewTicker and using Go channels. I don't have a lot of experience with channels, but understand high level how it works. I'm gonna play around with this to see it works as expected.

## Prometheus metrics

Before finishing the call out to services and parse, I want to start to skaffold the prometheus metrics to produce. Following this, it would be good to start setting up some simple docker compose with prom and grafana as well as a service to health check. I will go with elasticsearch because the status can be quite verbose and interesting for parsing, plus I know how to break it in an HA setup :D

## Timing/Alerting

I think I am just going to start with capturing the response times, and comparing this with the alerting threshold, but I would like to refactor this later to use a cancellable context to stop the request when reaching the threshold.


## Deployment

There are still a few things to improve on the checker side, but I want to complete the deployment side of things so it's done. I have a few ideas around the terraform/pulumi deploy, where I may include a hydrated Gitops solution if I make the minikube setup deterministic and easy to set up.

## Tests

Now that the implementation is mostly figured out and the behavior is not bound to change too much, I think it's a good time to write tests.

## Kind Cluster

Set up a kind cluster for the k8s deployment, but also to make it easier for someone else to recreate the same environment based on some configs I set.