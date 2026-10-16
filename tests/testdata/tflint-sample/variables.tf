variable "pingone_environment_id" {
  type        = string
  description = "The PingOne environment ID to configure DaVinci resources in"

  validation {
    condition     = can(regex("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$", var.pingone_environment_id))
    error_message = "The PingOne Environment ID must be a valid PingOne resource ID (UUID format)."
  }
}

# Pingone_davinci_connector_instance Variables

variable "davinci_connection_Code_0020_Snippet_code" {
  type        = string
  description = "code property for pingcli__Code-0020-Snippet connector"
}

variable "davinci_connection_Code_0020_Snippet_inputSchema" {
  type        = string
  description = "inputSchema property for pingcli__Code-0020-Snippet connector"
}

variable "davinci_connection_Apple_0020_Login_customAuth_providerName" {
  type        = string
  description = "providerName property for pingcli__Apple-0020-Login connector (customAuth)"
}

variable "davinci_connection_Apple_0020_Login_customAuth_clientSecret" {
  type        = string
  description = "clientSecret property for pingcli__Apple-0020-Login connector (customAuth)"
  sensitive   = true
}
