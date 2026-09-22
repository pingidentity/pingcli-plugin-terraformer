terraform {
  required_version = ">= 1.5"

  required_providers {
    pingone = {
      source  = "pingidentity/pingone"
      version = "1.16.0-beta.2"
    }
  }
}
