terraform {
  required_version = ">= 1.6.0"

  required_providers {
    helm = {
      source  = "hashicorp/helm"
      version = ">= 3.1.1"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = ">= 3.0.1"
    }
  }
}

# https://registry.terraform.io/providers/hashicorp/Helm/latest/docs#example-usage
provider "helm" {
  kubernetes = {
    config_path    = var.kubeconfig_path
    config_context = "kind-health-checker"
  }
}

resource "helm_release" "health_checker" {
  name             = "health-checker"
  namespace        = "health-checker"
  create_namespace = true

  chart             = abspath("${path.module}/../helm")
  dependency_update = true

  values = [
    file(abspath("${path.module}/../helm/values.yaml"))
  ]

  wait    = true
  timeout = 60
}
