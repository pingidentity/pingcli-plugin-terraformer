variable "environment_id" {
  description = "ID of the environment containing the custom resource."
  type        = string
}

variable "resource_id" {
  description = "ID of the custom resource to create this scope on."
  type        = string
}

resource "pingone_resource_scope" "e2e" {
  environment_id = var.environment_id
  resource_id    = var.resource_id
  name           = "pingcli-terraformer-e2e-scope"
  description    = "Custom resource scope fixture for the provisioned-environment E2E test."
}
