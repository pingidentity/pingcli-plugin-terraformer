terraform {
  required_version = ">= 1.5"

  required_providers {
    pingone = {
      source  = "pingidentity/pingone"
      version = ">= 1.19.1, < 2.0.0"
    }
  }
}

# Provider credentials come from PINGONE_CLIENT_ID / PINGONE_CLIENT_SECRET /
# PINGONE_ENVIRONMENT_ID / PINGONE_REGION_CODE environment variables. The
# provider's environment_id is the *worker* environment; the resources in
# ping-export-module target the managed environment via
# var.pingone_environment_id.
provider "pingone" {}

# State lives in S3 so GitHub Actions (regression-env-apply workflow) can
# repair the environment. Bucket/region/key/locking are supplied at init via
# -backend-config flags (see README.md "Adopting state").
terraform {
  backend "s3" {}
}
