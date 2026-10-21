variable "environment_id" {
  description = "ID of the environment to create these groups in."
  type        = string
}

resource "pingone_group" "e2e" {
  environment_id = var.environment_id
  name           = "pingcli-terraformer-e2e-group"
  description    = "Minimal group fixture for the provisioned-environment E2E test."
}

# Second group consumed by the group-nesting fixture module.
resource "pingone_group" "e2e_child" {
  environment_id = var.environment_id
  name           = "pingcli-terraformer-e2e-group-child"
  description    = "Child group consumed by the group-nesting fixture."
}
