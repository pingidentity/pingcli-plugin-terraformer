terraform {
  required_version = ">= 1.5"

  required_providers {
    pingone = {
      source  = "pingidentity/pingone"
      version = ">= 1.19.1, < 2.0.0"
    }
  }
}
