{ pkgs, lib, config, inputs, ... }:

{
  # https://devenv.sh/basics/
  env.GREET = "devenv";

  # https://devenv.sh/packages/
  packages = [ 
    pkgs.go
    pkgs.kubernetes-helm
    pkgs.kind
    pkgs.kubectl
    pkgs.k9s
    pkgs.opentofu
  ];

  # https://devenv.sh/scripts/
  scripts.hello.exec = ''
    echo hello from $GREET
  '';

  scripts.cluster-up.exec = ''
    if ! kind get clusters | grep -q health-checker; then
      kind create cluster --name health-checker --config k8s/kind-cluster.yaml
    fi
    kind export kubeconfig --name health-checker
  '';

  scripts.load-docker-image.exec = ''
    docker build -t health-checker:local .
    kind load docker-image health-checker:local --name health-checker
  '';


  scripts.helm-deploy.exec = ''
    helm upgrade --install health-checker helm/ --namespace health-checker --create-namespace
  '';

  scripts.elastic-test-yellow.exec = ''
    curl -X PUT "localhost:9200/test-yellow" -H 'Content-Type: application/json' -d '{
      "settings": {
        "number_of_shards": 1,
        "number_of_replicas": 1
      }
    }'
  '';

  scripts.elastic-cleanup.exec = ''
    curl -X DELETE "localhost:9200/test-yellow"
  '';

  # https://devenv.sh/basics/
  enterShell = ''
    hello         # Run scripts directly
    go version # Use packages
  '';

  # https://devenv.sh/tasks/
  # tasks = {
  #   "myproj:setup".exec = "mytool build";
  #   "devenv:enterShell".after = [ "myproj:setup" ];
  # };

  # https://devenv.sh/git-hooks/
  # git-hooks.hooks.shellcheck.enable = true;
  git-hooks.hooks.golangci-lint.enable = true;

  # See full reference at https://devenv.sh/reference/options/
}
