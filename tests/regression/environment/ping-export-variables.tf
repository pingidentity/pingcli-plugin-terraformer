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

variable "davinci_connection_Code_0020_Snippet_outputSchema" {
  type        = string
  description = "outputSchema property for pingcli__Code-0020-Snippet connector"
}

variable "davinci_connection_IDmission_0020___0020_OIDC_0020___0020_test_0020_clone_0020_93b5_customAuth_authTypeDropdown" {
  type        = string
  description = "authTypeDropdown property for pingcli__IDmission-0020---0020-OIDC-0020---0020-test-0020-clone-0020-93b5 connector (customAuth)"
}

variable "davinci_connection_IDmission_0020___0020_OIDC_0020___0020_test_0020_clone_0020_93b5_customAuth_clientId" {
  type        = string
  description = "clientId property for pingcli__IDmission-0020---0020-OIDC-0020---0020-test-0020-clone-0020-93b5 connector (customAuth)"
}

variable "davinci_connection_IDmission_0020___0020_OIDC_0020___0020_test_0020_clone_0020_93b5_customAuth_clientSecret" {
  type        = string
  description = "clientSecret property for pingcli__IDmission-0020---0020-OIDC-0020---0020-test-0020-clone-0020-93b5 connector (customAuth)"
  sensitive   = true
}

variable "davinci_connection_IDmission_0020___0020_OIDC_0020___0020_test_0020_clone_0020_93b5_customAuth_providerName" {
  type        = string
  description = "providerName property for pingcli__IDmission-0020---0020-OIDC-0020---0020-test-0020-clone-0020-93b5 connector (customAuth)"
}

variable "davinci_connection_IDmission_0020___0020_OIDC_0020___0020_test_0020_clone_0020_93b5_customAuth_scope" {
  type        = string
  description = "scope property for pingcli__IDmission-0020---0020-OIDC-0020---0020-test-0020-clone-0020-93b5 connector (customAuth)"
}

variable "davinci_connection_IDmission_0020___0020_OIDC_0020___0020_test_0020_clone_0020_93b5_customAuth_state" {
  type        = string
  description = "state property for pingcli__IDmission-0020---0020-OIDC-0020---0020-test-0020-clone-0020-93b5 connector (customAuth)"
}

variable "davinci_connection_IDmission_0020___0020_OIDC_0020___0020_test_customAuth_authTypeDropdown" {
  type        = string
  description = "authTypeDropdown property for pingcli__IDmission-0020---0020-OIDC-0020---0020-test connector (customAuth)"
}

variable "davinci_connection_IDmission_0020___0020_OIDC_0020___0020_test_customAuth_clientId" {
  type        = string
  description = "clientId property for pingcli__IDmission-0020---0020-OIDC-0020---0020-test connector (customAuth)"
}

variable "davinci_connection_IDmission_0020___0020_OIDC_0020___0020_test_customAuth_clientSecret" {
  type        = string
  description = "clientSecret property for pingcli__IDmission-0020---0020-OIDC-0020---0020-test connector (customAuth)"
  sensitive   = true
}

variable "davinci_connection_IDmission_0020___0020_OIDC_0020___0020_test_customAuth_providerName" {
  type        = string
  description = "providerName property for pingcli__IDmission-0020---0020-OIDC-0020---0020-test connector (customAuth)"
}

variable "davinci_connection_IDmission_0020___0020_OIDC_0020___0020_test_customAuth_scope" {
  type        = string
  description = "scope property for pingcli__IDmission-0020---0020-OIDC-0020---0020-test connector (customAuth)"
}

variable "davinci_connection_IDmission_0020___0020_OIDC_0020___0020_test_customAuth_state" {
  type        = string
  description = "state property for pingcli__IDmission-0020---0020-OIDC-0020---0020-test connector (customAuth)"
}

variable "davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020___0020_test_customAuth_clientSecret" {
  type        = string
  description = "clientSecret property for pingcli__OIDC-0020--0026--0020-OAuth-0020-IdP-0020---0020-test connector (customAuth)"
  sensitive   = true
}

variable "davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020___0020_test_customAuth_customAttributes" {
  type        = string
  description = "customAttributes property for pingcli__OIDC-0020--0026--0020-OAuth-0020-IdP-0020---0020-test connector (customAuth)"
}

variable "davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020___0020_test_customAuth_state" {
  type        = string
  description = "state property for pingcli__OIDC-0020--0026--0020-OAuth-0020-IdP-0020---0020-test connector (customAuth)"
}

variable "davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testB_customAuth_authorizationEndpoint" {
  type        = string
  description = "authorizationEndpoint property for pingcli__OIDC-0020--0026--0020-OAuth-0020-IdP-0020--testB connector (customAuth)"
}

variable "davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testB_customAuth_authTypeDropdown" {
  type        = string
  description = "authTypeDropdown property for pingcli__OIDC-0020--0026--0020-OAuth-0020-IdP-0020--testB connector (customAuth)"
}

variable "davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testB_customAuth_clientId" {
  type        = string
  description = "clientId property for pingcli__OIDC-0020--0026--0020-OAuth-0020-IdP-0020--testB connector (customAuth)"
}

variable "davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testB_customAuth_clientSecret" {
  type        = string
  description = "clientSecret property for pingcli__OIDC-0020--0026--0020-OAuth-0020-IdP-0020--testB connector (customAuth)"
  sensitive   = true
}

variable "davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testB_customAuth_customAttributes" {
  type        = string
  description = "customAttributes property for pingcli__OIDC-0020--0026--0020-OAuth-0020-IdP-0020--testB connector (customAuth)"
}

variable "davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testB_customAuth_providerName" {
  type        = string
  description = "providerName property for pingcli__OIDC-0020--0026--0020-OAuth-0020-IdP-0020--testB connector (customAuth)"
}

variable "davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testB_customAuth_scope" {
  type        = string
  description = "scope property for pingcli__OIDC-0020--0026--0020-OAuth-0020-IdP-0020--testB connector (customAuth)"
}

variable "davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testB_customAuth_tokenEndpoint" {
  type        = string
  description = "tokenEndpoint property for pingcli__OIDC-0020--0026--0020-OAuth-0020-IdP-0020--testB connector (customAuth)"
}

variable "davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testB_customAuth_userInfoEndpoint" {
  type        = string
  description = "userInfoEndpoint property for pingcli__OIDC-0020--0026--0020-OAuth-0020-IdP-0020--testB connector (customAuth)"
}

variable "davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testC_customAuth_authorizationEndpoint" {
  type        = string
  description = "authorizationEndpoint property for pingcli__OIDC-0020--0026--0020-OAuth-0020-IdP-0020--testC connector (customAuth)"
}

variable "davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testC_customAuth_authTypeDropdown" {
  type        = string
  description = "authTypeDropdown property for pingcli__OIDC-0020--0026--0020-OAuth-0020-IdP-0020--testC connector (customAuth)"
}

variable "davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testC_customAuth_clientId" {
  type        = string
  description = "clientId property for pingcli__OIDC-0020--0026--0020-OAuth-0020-IdP-0020--testC connector (customAuth)"
}

variable "davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testC_customAuth_clientSecret" {
  type        = string
  description = "clientSecret property for pingcli__OIDC-0020--0026--0020-OAuth-0020-IdP-0020--testC connector (customAuth)"
  sensitive   = true
}

variable "davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testC_customAuth_customAttributes" {
  type        = string
  description = "customAttributes property for pingcli__OIDC-0020--0026--0020-OAuth-0020-IdP-0020--testC connector (customAuth)"
}

variable "davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testC_customAuth_providerName" {
  type        = string
  description = "providerName property for pingcli__OIDC-0020--0026--0020-OAuth-0020-IdP-0020--testC connector (customAuth)"
}

variable "davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testC_customAuth_scope" {
  type        = string
  description = "scope property for pingcli__OIDC-0020--0026--0020-OAuth-0020-IdP-0020--testC connector (customAuth)"
}

variable "davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testC_customAuth_tokenEndpoint" {
  type        = string
  description = "tokenEndpoint property for pingcli__OIDC-0020--0026--0020-OAuth-0020-IdP-0020--testC connector (customAuth)"
}

variable "davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testC_customAuth_userInfoEndpoint" {
  type        = string
  description = "userInfoEndpoint property for pingcli__OIDC-0020--0026--0020-OAuth-0020-IdP-0020--testC connector (customAuth)"
}

variable "davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testD_customAuth_authorizationEndpoint" {
  type        = string
  description = "authorizationEndpoint property for pingcli__OIDC-0020--0026--0020-OAuth-0020-IdP-0020--testD connector (customAuth)"
}

variable "davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testD_customAuth_authTypeDropdown" {
  type        = string
  description = "authTypeDropdown property for pingcli__OIDC-0020--0026--0020-OAuth-0020-IdP-0020--testD connector (customAuth)"
}

variable "davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testD_customAuth_clientId" {
  type        = string
  description = "clientId property for pingcli__OIDC-0020--0026--0020-OAuth-0020-IdP-0020--testD connector (customAuth)"
}

variable "davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testD_customAuth_clientSecret" {
  type        = string
  description = "clientSecret property for pingcli__OIDC-0020--0026--0020-OAuth-0020-IdP-0020--testD connector (customAuth)"
  sensitive   = true
}

variable "davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testD_customAuth_customAttributes" {
  type        = string
  description = "customAttributes property for pingcli__OIDC-0020--0026--0020-OAuth-0020-IdP-0020--testD connector (customAuth)"
}

variable "davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testD_customAuth_providerName" {
  type        = string
  description = "providerName property for pingcli__OIDC-0020--0026--0020-OAuth-0020-IdP-0020--testD connector (customAuth)"
}

variable "davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testD_customAuth_scope" {
  type        = string
  description = "scope property for pingcli__OIDC-0020--0026--0020-OAuth-0020-IdP-0020--testD connector (customAuth)"
}

variable "davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testD_customAuth_tokenEndpoint" {
  type        = string
  description = "tokenEndpoint property for pingcli__OIDC-0020--0026--0020-OAuth-0020-IdP-0020--testD connector (customAuth)"
}

variable "davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testD_customAuth_userInfoEndpoint" {
  type        = string
  description = "userInfoEndpoint property for pingcli__OIDC-0020--0026--0020-OAuth-0020-IdP-0020--testD connector (customAuth)"
}

variable "davinci_connection_PingOne_0020_MFA_clientId" {
  type        = string
  description = "clientId property for pingcli__PingOne-0020-MFA connector"
}

variable "davinci_connection_PingOne_0020_MFA_clientSecret" {
  type        = string
  description = "clientSecret property for pingcli__PingOne-0020-MFA connector"
  sensitive   = true
}

variable "davinci_connection_PingOne_0020_MFA_envId" {
  type        = string
  description = "envId property for pingcli__PingOne-0020-MFA connector"
}

variable "davinci_connection_PingOne_0020_MFA_policyId" {
  type        = string
  description = "policyId property for pingcli__PingOne-0020-MFA connector"
}

variable "davinci_connection_PingOne_0020_MFA_region" {
  type        = string
  description = "region property for pingcli__PingOne-0020-MFA connector"
}

variable "davinci_connection_PingOne_0020_Notifications_clientId" {
  type        = string
  description = "clientId property for pingcli__PingOne-0020-Notifications connector"
}

variable "davinci_connection_PingOne_0020_Notifications_clientSecret" {
  type        = string
  description = "clientSecret property for pingcli__PingOne-0020-Notifications connector"
  sensitive   = true
}

variable "davinci_connection_PingOne_0020_Notifications_envId" {
  type        = string
  description = "envId property for pingcli__PingOne-0020-Notifications connector"
}

variable "davinci_connection_PingOne_0020_Notifications_region" {
  type        = string
  description = "region property for pingcli__PingOne-0020-Notifications connector"
}

variable "davinci_connection_PingOne_0020_Protect_clientId" {
  type        = string
  description = "clientId property for pingcli__PingOne-0020-Protect connector"
}

variable "davinci_connection_PingOne_0020_Protect_clientSecret" {
  type        = string
  description = "clientSecret property for pingcli__PingOne-0020-Protect connector"
  sensitive   = true
}

variable "davinci_connection_PingOne_0020_Protect_envId" {
  type        = string
  description = "envId property for pingcli__PingOne-0020-Protect connector"
}

variable "davinci_connection_PingOne_0020_Protect_region" {
  type        = string
  description = "region property for pingcli__PingOne-0020-Protect connector"
}

variable "davinci_connection_PingOne_clientId" {
  type        = string
  description = "clientId property for pingcli__PingOne connector"
}

variable "davinci_connection_PingOne_clientSecret" {
  type        = string
  description = "clientSecret property for pingcli__PingOne connector"
  sensitive   = true
}

variable "davinci_connection_PingOne_envId" {
  type        = string
  description = "envId property for pingcli__PingOne connector"
}

variable "davinci_connection_PingOne_region" {
  type        = string
  description = "region property for pingcli__PingOne connector"
}

# Pingone_davinci_form Variables

variable "davinci_form_example___sign_on" {
  type        = string
  description = "ID for pingone_davinci_form resource (not yet exported)"
}

# Pingone_davinci_variable Variables

variable "davinci_variable_buttonValue_value" {
  type        = string
  description = "Value for DaVinci variable buttonValue"
}

variable "davinci_variable_ciam_accountRecoveryEnabled_value" {
  type        = string
  description = "Value for DaVinci variable ciam_accountRecoveryEnabled"
}

variable "davinci_variable_ciam_agreementEnabled_value" {
  type        = string
  description = "Value for DaVinci variable ciam_agreementEnabled"
}

variable "davinci_variable_ciam_appleEnabled_value" {
  type        = string
  description = "Value for DaVinci variable ciam_appleEnabled"
}

variable "davinci_variable_ciam_companyName_value" {
  type        = string
  description = "Value for DaVinci variable ciam_companyName"
}

variable "davinci_variable_ciam_emailOtpEnabled_value" {
  type        = string
  description = "Value for DaVinci variable ciam_emailOtpEnabled"
}

variable "davinci_variable_ciam_facebookEnabled_value" {
  type        = string
  description = "Value for DaVinci variable ciam_facebookEnabled"
}

variable "davinci_variable_ciam_fidoPasskeyEnabled_value" {
  type        = string
  description = "Value for DaVinci variable ciam_fidoPasskeyEnabled"
}

variable "davinci_variable_ciam_googleEnabled_value" {
  type        = string
  description = "Value for DaVinci variable ciam_googleEnabled"
}

variable "davinci_variable_ciam_logoStyle_value" {
  type        = string
  description = "Value for DaVinci variable ciam_logoStyle"
}

variable "davinci_variable_ciam_logoUrl_value" {
  type        = string
  description = "Value for DaVinci variable ciam_logoUrl"
}

variable "davinci_variable_ciam_magicLinkEnabled_value" {
  type        = string
  description = "Value for DaVinci variable ciam_magicLinkEnabled"
}

variable "davinci_variable_ciam_mobilePushOtpEnabled_value" {
  type        = string
  description = "Value for DaVinci variable ciam_mobilePushOtpEnabled"
}

variable "davinci_variable_ciam_otpFallbackAllowed_value" {
  type        = string
  description = "Value for DaVinci variable ciam_otpFallbackAllowed"
}

variable "davinci_variable_ciam_passwordlessRequired_value" {
  type        = string
  description = "Value for DaVinci variable ciam_passwordlessRequired"
}

variable "davinci_variable_ciam_recoveryLimit_value" {
  type        = string
  description = "Value for DaVinci variable ciam_recoveryLimit"
}

variable "davinci_variable_ciam_requireMFA_value" {
  type        = string
  description = "Value for DaVinci variable ciam_requireMFA"
}

variable "davinci_variable_ciam_resendOtpLimit_value" {
  type        = string
  description = "Value for DaVinci variable ciam_resendOtpLimit"
}

variable "davinci_variable_ciam_sessionLengthInMinute_value" {
  type        = string
  description = "Value for DaVinci variable ciam_sessionLengthInMinute"
}

variable "davinci_variable_ciam_smsOtpEnabled_value" {
  type        = string
  description = "Value for DaVinci variable ciam_smsOtpEnabled"
}

variable "davinci_variable_ciam_totpEnabled_value" {
  type        = string
  description = "Value for DaVinci variable ciam_totpEnabled"
}

variable "davinci_variable_ciam_verificationLimit_value" {
  type        = string
  description = "Value for DaVinci variable ciam_verificationLimit"
}

variable "davinci_variable_ciam_voiceOtpEnabled_value" {
  type        = string
  description = "Value for DaVinci variable ciam_voiceOtpEnabled"
}

variable "davinci_variable_companyContextVar_value" {
  type        = string
  description = "Value for DaVinci variable companyContextVar"
}

variable "davinci_variable_flowRequireMFA_value" {
  type        = string
  description = "Value for DaVinci variable flowRequireMFA"
}

variable "davinci_variable_hideSkipButton_value" {
  type        = string
  description = "Value for DaVinci variable hideSkipButton"
}

variable "davinci_variable_name_value" {
  type        = string
  description = "Value for DaVinci variable name"
}

variable "davinci_variable_registrationAgreementId_value" {
  type        = string
  description = "Value for DaVinci variable registrationAgreementId"
}

variable "davinci_variable_registrationPopulationId_value" {
  type        = string
  description = "Value for DaVinci variable registrationPopulationId"
}

variable "davinci_variable_testPassword_value" {
  type        = string
  description = "Value for DaVinci variable testPassword"
}

variable "davinci_variable_testPW_value" {
  type        = string
  description = "Value for DaVinci variable testPW"
}

