variable "environment_id" {
  description = "ID of the environment containing the application."
  type        = string
}

variable "application_id" {
  description = "ID of the application to map the attribute on."
  type        = string
}

# The name must avoid reserved OIDC claim names (sub, aud, ...). The value
# must be a ${user.<attr>} SCIM expression ($${ escapes it from Terraform
# template interpolation).
resource "pingone_application_attribute_mapping" "e2e" {
  environment_id = var.environment_id
  application_id = var.application_id
  name           = "pingcli-terraformer-e2e-mapping"
  value          = "$${user.id}"
}
