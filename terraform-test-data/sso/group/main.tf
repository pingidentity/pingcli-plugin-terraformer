variable "environment_id" {
  description = "ID of the environment to create this group in."
  type        = string
}

resource "pingone_group" "e2e" {
  environment_id = var.environment_id
  name           = "pingcli-terraformer-e2e-group"
  description    = "Minimal group fixture for the provisioned-environment E2E test."
}
