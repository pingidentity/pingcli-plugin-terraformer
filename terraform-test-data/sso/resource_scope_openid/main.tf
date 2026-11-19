variable "environment_id" {
  description = "ID of the environment to create this OIDC scope in."
  type        = string
}

# The built-in OPENID_CONNECT resource's ID is computed (one exists per
# environment), so only environment_id and the scope name are inputs.
# Conservative alphanumeric name — the provider's accepted character set for
# OIDC scope names is not documented.
resource "pingone_resource_scope_openid" "e2e" {
  environment_id = var.environment_id
  name           = "pingclie2eoidcscope"
  description    = "OIDC resource scope fixture for the provisioned-environment E2E test."
}
