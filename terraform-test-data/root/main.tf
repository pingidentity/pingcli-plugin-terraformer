# Composition root for terraform-test-data fixtures. Applied by
# tools/tf-regression-provision using org-admin credentials, which both
# create the throwaway environment (pingone_environment below) and
# authenticate every fixture resource inside it - one provider config, one
# dependency graph, one state. Fixture modules get added here in dependency
# order as terraform-test-data grows (see contributing docs for the rollout
# sequencing).

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
