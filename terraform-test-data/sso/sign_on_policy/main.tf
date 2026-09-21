variable "environment_id" {
  description = "ID of the environment to create this sign-on policy in."
  type        = string
}

resource "pingone_sign_on_policy" "e2e" {
  environment_id = var.environment_id
  name           = "pingcli-terraformer-e2e-sign-on-policy"
  description    = "Minimal sign-on policy fixture for the provisioned-environment E2E test."
}
