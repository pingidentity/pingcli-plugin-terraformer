variable "environment_id" {
  description = "ID of the environment containing the custom resource."
  type        = string
}

variable "custom_resource_id" {
  description = "ID of the custom resource this attribute belongs to."
  type        = string
}

# resource_type = "CUSTOM" is required by the provider when custom_resource_id
# is set. The value must be a ${user.<attr>} SCIM expression ($${ escapes it
# from Terraform template interpolation). The exporter only exports
# CUSTOM-type attributes, so this fixture is what the attribute export covers.
resource "pingone_resource_attribute" "e2e" {
  environment_id     = var.environment_id
  resource_type      = "CUSTOM"
  custom_resource_id = var.custom_resource_id
  name               = "pingcli-terraformer-e2e-attribute"
  value              = "$${user.id}"
}
