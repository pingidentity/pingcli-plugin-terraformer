variable "environment_id" {
  description = "ID of the environment to create this password policy in."
  type        = string
}

# Nested policy blocks (history, length, lockout, min_characters) are all
# optional — the provider applies its own defaults when omitted, so a minimal
# block is valid.
resource "pingone_password_policy" "e2e" {
  environment_id = var.environment_id
  name           = "pingcli-terraformer-e2e-password-policy"
  description    = "Minimal password policy fixture for the provisioned-environment E2E test."
}
