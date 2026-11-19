variable "environment_id" {
  description = "ID of the environment to create this sign-on policy in."
  type        = string
}

resource "pingone_sign_on_policy" "e2e" {
  environment_id = var.environment_id
  name           = "pingcli-terraformer-e2e-sign-on-policy"
  description    = "Minimal sign-on policy fixture for the provisioned-environment E2E test."
}

# The PingOne API refuses to assign a policy with no actions, so the policy
# carries a single login action. pingone_sign_on_policy_action is not
# exported by the exporter (no definition/handler) — it exists here purely to
# make the policy assignable by the sign-on-policy-assignment fixture.
resource "pingone_sign_on_policy_action" "e2e_login" {
  environment_id    = var.environment_id
  sign_on_policy_id = pingone_sign_on_policy.e2e.id
  priority          = 1

  login {}
}
