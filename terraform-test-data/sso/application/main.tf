variable "environment_id" {
  description = "ID of the environment to create this application in."
  type        = string
}

# Minimal OIDC WEB_APP application. application_secret coverage is automatic:
# PingOne generates a secret on creation and the exporter reads it as a 1:1
# singleton alongside this application.
resource "pingone_application" "e2e" {
  environment_id = var.environment_id
  name           = "pingcli-terraformer-e2e-application"
  description    = "Minimal OIDC application fixture for the provisioned-environment E2E test."

  # oidc_options is a single-nested-object ATTRIBUTE in the provider schema
  # (assigned with "=", not a block). response_types must include "CODE" when
  # grant_types contains AUTHORIZATION_CODE — the PingOne API rejects the
  # create otherwise (a live-API validation terraform validate cannot check).
  oidc_options = {
    type                       = "WEB_APP"
    grant_types                = ["AUTHORIZATION_CODE", "REFRESH_TOKEN"]
    response_types             = ["CODE"]
    token_endpoint_auth_method = "CLIENT_SECRET_BASIC"
    redirect_uris              = ["https://pingcli-terraformer-e2e.invalid/callback"]
  }
}
