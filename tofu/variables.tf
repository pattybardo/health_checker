variable "kubeconfig_path" {
  type        = string
  description = "Path to the kubeconfig used to reach your kind cluster."
  default     = "~/.kube/config"
}
