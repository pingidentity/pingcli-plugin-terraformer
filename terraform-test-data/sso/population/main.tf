variable "environment_id" {
  description = "ID of the environment to create this population in."
  type        = string
}

resource "pingone_population" "e2e" {
  environment_id = var.environment_id
  name           = "pingcli-terraformer-e2e-population"
  description    = "Minimal population fixture for the provisioned-environment E2E test."
}
