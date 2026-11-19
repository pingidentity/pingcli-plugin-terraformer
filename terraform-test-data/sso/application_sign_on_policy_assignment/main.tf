variable "environment_id" {
  description = "ID of the environment containing the application and policy."
  type        = string
}

variable "application_id" {
  description = "ID of the application to assign the sign-on policy to."
  type        = string
}

variable "sign_on_policy_id" {
  description = "ID of the sign-on policy to assign. Deliberately the fixture policy, not the environment default (PingOne reserves the default)."
  type        = string
}

resource "pingone_application_sign_on_policy_assignment" "e2e" {
  environment_id    = var.environment_id
  application_id    = var.application_id
  sign_on_policy_id = var.sign_on_policy_id
  priority          = 1
}
