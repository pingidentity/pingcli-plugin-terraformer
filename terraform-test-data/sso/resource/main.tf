variable "environment_id" {
  description = "ID of the environment to create this resource in."
  type        = string
}

resource "pingone_resource" "e2e" {
  environment_id = var.environment_id
  name           = "pingcli-terraformer-e2e-resource"
  description    = "Minimal custom resource fixture for the provisioned-environment E2E test."
}
