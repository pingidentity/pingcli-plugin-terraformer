# Composition root for terraform-test-data fixtures. Applied by
# tools/tf-regression-provision using org-admin credentials, which both
# create the throwaway environment (pingone_environment below) and
# authenticate every fixture resource inside it - one provider config, one
# dependency graph, one state. Fixture modules get added here in dependency
# order as terraform-test-data grows (see tests/regression-provisioned/README.md
# for the fixture-module contract and rollout sequencing).

provider "pingone" {
  environment_id = var.org_admin_environment_id
  client_id      = var.org_admin_client_id
  client_secret  = var.org_admin_client_secret
  region_code    = var.region_code
}

resource "pingone_environment" "this" {
  name        = var.environment_name
  description = "Throwaway environment for the pingcli-plugin-terraformer provisioned-environment E2E test. Safe to delete."
  type        = "SANDBOX"
  license_id  = var.license_id

  services = [
    { type = "SSO" },
  ]
}

module "sso_population" {
  source         = "../sso/population"
  environment_id = pingone_environment.this.id
}

module "sso_group" {
  source         = "../sso/group"
  environment_id = pingone_environment.this.id
}

module "sso_group_nesting" {
  source          = "../sso/group_nesting"
  environment_id  = pingone_environment.this.id
  parent_group_id = module.sso_group.parent_group_id
  child_group_id  = module.sso_group.child_group_id
}

module "sso_sign_on_policy" {
  source         = "../sso/sign_on_policy"
  environment_id = pingone_environment.this.id
}

module "sso_password_policy" {
  source         = "../sso/password_policy"
  environment_id = pingone_environment.this.id
}

module "sso_population_default" {
  source         = "../sso/population_default"
  environment_id = pingone_environment.this.id
}

module "sso_resource" {
  source         = "../sso/resource"
  environment_id = pingone_environment.this.id
}

module "sso_resource_scope" {
  source         = "../sso/resource_scope"
  environment_id = pingone_environment.this.id
  resource_id    = module.sso_resource.resource_id
}

module "sso_resource_attribute" {
  source             = "../sso/resource_attribute"
  environment_id     = pingone_environment.this.id
  custom_resource_id = module.sso_resource.resource_id
}

module "sso_resource_scope_openid" {
  source         = "../sso/resource_scope_openid"
  environment_id = pingone_environment.this.id
}

module "sso_resource_scope_pingone_api" {
  source         = "../sso/resource_scope_pingone_api"
  environment_id = pingone_environment.this.id
}

module "sso_application" {
  source         = "../sso/application"
  environment_id = pingone_environment.this.id
}

module "sso_application_sign_on_policy_assignment" {
  source            = "../sso/application_sign_on_policy_assignment"
  environment_id    = pingone_environment.this.id
  application_id    = module.sso_application.application_id
  sign_on_policy_id = module.sso_sign_on_policy.sign_on_policy_id
}

module "sso_application_attribute_mapping" {
  source         = "../sso/application_attribute_mapping"
  environment_id = pingone_environment.this.id
  application_id = module.sso_application.application_id
}
