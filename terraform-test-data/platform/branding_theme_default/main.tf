variable "environment_id" {
  description = "ID of the environment whose default branding theme this fixture activates."
  type        = string
}

variable "branding_theme_id" {
  description = "ID of the branding theme to activate as the environment default."
  type        = string
}

# pingone_branding_theme_default is an environment-scoped singleton: exactly
# one per environment (the API models it as the `default` flag on the theme;
# the provider as PUT .../branding/themes/{themeID}/default). The fixture
# theme itself lives at the root (terraform-test-data/root/main.tf) so the
# environment's branding stays visible alongside the composition, and the
# default-activation fixture consumes its ID via the output→input pattern.
resource "pingone_branding_theme_default" "e2e" {
  environment_id    = var.environment_id
  branding_theme_id = var.branding_theme_id
}
