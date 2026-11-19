variable "org_admin_environment_id" {
  description = "Home environment of the org-admin worker application used to authenticate the provider."
  type        = string
}

variable "org_admin_client_id" {
  description = "Client ID of the org-admin worker application."
  type        = string
}

variable "org_admin_client_secret" {
  description = "Client secret of the org-admin worker application."
  type        = string
  sensitive   = true
}

variable "region_code" {
  description = "PingOne region code (NA, EU, AP, CA, AU, SG)."
  type        = string
}

variable "license_id" {
  description = "License ID to assign to the throwaway environment created by this run."
  type        = string
}

variable "environment_name" {
  description = "Name for the throwaway environment created by this run."
  type        = string
}
