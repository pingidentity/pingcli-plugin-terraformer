variable "environment_id" {
  description = "ID of the environment to create this PingOne API sub-scope in."
  type        = string
}

# The name must satisfy the provider validator ^p1:(read|update):user(:...)*$.
# The ":e2e" suffix keeps it distinct from the built-in p1:read:user /
# p1:update:user scopes, and satisfies the exporter's identical silent filter
# (built-in non-user p1:* scopes are filtered from the export, so the fixture
# name must land inside the exported subset to provide coverage).
resource "pingone_resource_scope_pingone_api" "e2e" {
  environment_id = var.environment_id
  name           = "p1:read:user:e2e"
  description    = "PingOne API sub-scope fixture for the provisioned-environment E2E test."
}
