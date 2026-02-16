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

  scripts.cluster-load.exec = ''
    docker build -t health-checker:local .
    kind load docker-image health-checker:local --name health-checker
  '';


  scripts.helm-deploy.exec = ''
    helm upgrade --install health-checker helm/ --namespace health-checker --create-namespace
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
