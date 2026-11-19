variable "environment_id" {
  description = "ID of the environment whose default population this fixture renames."
  type        = string
}

# pingone_population_default is an environment-scoped singleton: on create the
# provider adopts (and renames) the environment's auto-created default
# population rather than making a second one. Import id is the environment ID.
resource "pingone_population_default" "e2e" {
  environment_id = var.environment_id
  name           = "pingcli-terraformer-e2e-default-population"
  description    = "Renamed default population fixture for the provisioned-environment E2E test."
}
