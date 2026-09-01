

resource "pingone_davinci_variable" "pingcli__agreementId_flowInstance" {
  environment_id = var.pingone_environment_id
  name           = "agreementId"
  context        = "flowInstance"
  data_type      = "string"
  mutable        = true
  min            = 0
  max            = 2000
}


resource "pingone_davinci_variable" "pingcli__buttonValue_flowInstance" {
  environment_id = var.pingone_environment_id
  name           = "buttonValue"
  context        = "flowInstance"
  data_type      = "string"
  mutable        = true
  display_name   = "simulate agreement button selection"
  min            = 0
  max            = 2000

  value = {
    string = var.davinci_variable_buttonValue_value
  }
}


resource "pingone_davinci_variable" "pingcli__cachedEmail_flowInstance" {
  environment_id = var.pingone_environment_id
  name           = "cachedEmail"
  context        = "flowInstance"
  data_type      = "string"
  mutable        = true
  min            = 0
  max            = 2000
}


resource "pingone_davinci_variable" "pingcli__ciam_accountRecoveryEnabled_company" {
  environment_id = var.pingone_environment_id
  name           = "ciam_accountRecoveryEnabled"
  context        = "company"
  data_type      = "boolean"
  mutable        = true
  min            = 0
  max            = 2000

  value = {
    string = var.davinci_variable_ciam_accountRecoveryEnabled_value
  }
}


resource "pingone_davinci_variable" "pingcli__ciam_agreementEnabled_company" {
  environment_id = var.pingone_environment_id
  name           = "ciam_agreementEnabled"
  context        = "company"
  data_type      = "boolean"
  mutable        = true
  min            = 0
  max            = 2000

  value = {
    string = var.davinci_variable_ciam_agreementEnabled_value
  }
}


resource "pingone_davinci_variable" "pingcli__ciam_agreementID_flowInstance" {
  environment_id = var.pingone_environment_id
  name           = "ciam_agreementID"
  context        = "flowInstance"
  data_type      = "string"
  mutable        = true
  min            = 0
  max            = 2000
}


resource "pingone_davinci_variable" "pingcli__ciam_appleEnabled_company" {
  environment_id = var.pingone_environment_id
  name           = "ciam_appleEnabled"
  context        = "company"
  data_type      = "boolean"
  mutable        = true
  min            = 0
  max            = 2000

  value = {
    string = var.davinci_variable_ciam_appleEnabled_value
  }
}


resource "pingone_davinci_variable" "pingcli__ciam_authMethod_flowInstance" {
  environment_id = var.pingone_environment_id
  name           = "ciam_authMethod"
  context        = "flowInstance"
  data_type      = "string"
  mutable        = true
  min            = 0
  max            = 2000
}


resource "pingone_davinci_variable" "pingcli__ciam_companyName_company" {
  environment_id = var.pingone_environment_id
  name           = "ciam_companyName"
  context        = "company"
  data_type      = "string"
  mutable        = false
  min            = 0
  max            = 2000

  value = {
    string = var.davinci_variable_ciam_companyName_value
  }
}


resource "pingone_davinci_variable" "pingcli__ciam_deviceAuthnID_flowInstance" {
  environment_id = var.pingone_environment_id
  name           = "ciam_deviceAuthnID"
  context        = "flowInstance"
  data_type      = "string"
  mutable        = true
  min            = 0
  max            = 2000
}


resource "pingone_davinci_variable" "pingcli__ciam_deviceId_flowInstance" {
  environment_id = var.pingone_environment_id
  name           = "ciam_deviceId"
  context        = "flowInstance"
  data_type      = "string"
  mutable        = true
  min            = 0
  max            = 2000
}


resource "pingone_davinci_variable" "pingcli__ciam_emailOtpEnabled_company" {
  environment_id = var.pingone_environment_id
  name           = "ciam_emailOtpEnabled"
  context        = "company"
  data_type      = "boolean"
  mutable        = true
  min            = 0
  max            = 2000

  value = {
    string = var.davinci_variable_ciam_emailOtpEnabled_value
  }
}


resource "pingone_davinci_variable" "pingcli__ciam_errorMessage_flowInstance" {
  environment_id = var.pingone_environment_id
  name           = "ciam_errorMessage"
  context        = "flowInstance"
  data_type      = "string"
  mutable        = true
  min            = 0
  max            = 2000
}


resource "pingone_davinci_variable" "pingcli__ciam_facebookEnabled_company" {
  environment_id = var.pingone_environment_id
  name           = "ciam_facebookEnabled"
  context        = "company"
  data_type      = "boolean"
  mutable        = true
  min            = 0
  max            = 2000

  value = {
    string = var.davinci_variable_ciam_facebookEnabled_value
  }
}


resource "pingone_davinci_variable" "pingcli__ciam_fidoPasskeyEnabled_company" {
  environment_id = var.pingone_environment_id
  name           = "ciam_fidoPasskeyEnabled"
  context        = "company"
  data_type      = "boolean"
  mutable        = true
  min            = 0
  max            = 2000

  value = {
    string = var.davinci_variable_ciam_fidoPasskeyEnabled_value
  }
}


resource "pingone_davinci_variable" "pingcli__ciam_googleEnabled_company" {
  environment_id = var.pingone_environment_id
  name           = "ciam_googleEnabled"
  context        = "company"
  data_type      = "boolean"
  mutable        = true
  min            = 0
  max            = 2000

  value = {
    string = var.davinci_variable_ciam_googleEnabled_value
  }
}


resource "pingone_davinci_variable" "pingcli__ciam_logoStyle_company" {
  environment_id = var.pingone_environment_id
  name           = "ciam_logoStyle"
  context        = "company"
  data_type      = "string"
  mutable        = true
  display_name   = "CSS style for company logo"
  min            = 0
  max            = 2000

  value = {
    string = var.davinci_variable_ciam_logoStyle_value
  }
}


resource "pingone_davinci_variable" "pingcli__ciam_logoUrl_company" {
  environment_id = var.pingone_environment_id
  name           = "ciam_logoUrl"
  context        = "company"
  data_type      = "string"
  mutable        = true
  display_name   = "URL of company logo"
  min            = 0
  max            = 2000

  value = {
    string = var.davinci_variable_ciam_logoUrl_value
  }
}


resource "pingone_davinci_variable" "pingcli__ciam_magicLinkEnabled_company" {
  environment_id = var.pingone_environment_id
  name           = "ciam_magicLinkEnabled"
  context        = "company"
  data_type      = "boolean"
  mutable        = true
  min            = 0
  max            = 2000

  value = {
    string = var.davinci_variable_ciam_magicLinkEnabled_value
  }
}


resource "pingone_davinci_variable" "pingcli__ciam_mfaPolicyID_flowInstance" {
  environment_id = var.pingone_environment_id
  name           = "ciam_mfaPolicyID"
  context        = "flowInstance"
  data_type      = "string"
  mutable        = true
  min            = 0
  max            = 2000
}


resource "pingone_davinci_variable" "pingcli__ciam_mobilePushOtpEnabled_company" {
  environment_id = var.pingone_environment_id
  name           = "ciam_mobilePushOtpEnabled"
  context        = "company"
  data_type      = "boolean"
  mutable        = true
  min            = 0
  max            = 2000

  value = {
    string = var.davinci_variable_ciam_mobilePushOtpEnabled_value
  }
}


resource "pingone_davinci_variable" "pingcli__ciam_otpFallbackAllowed_company" {
  environment_id = var.pingone_environment_id
  name           = "ciam_otpFallbackAllowed"
  context        = "company"
  data_type      = "boolean"
  mutable        = true
  display_name   = "When true a timed out a mobile application push request should fall to prompt OTP in flow\t"
  min            = 0
  max            = 2000

  value = {
    string = var.davinci_variable_ciam_otpFallbackAllowed_value
  }
}


resource "pingone_davinci_variable" "pingcli__ciam_passwordlessRequired_company" {
  environment_id = var.pingone_environment_id
  name           = "ciam_passwordlessRequired"
  context        = "company"
  data_type      = "boolean"
  mutable        = false
  min            = 0
  max            = 2000

  value = {
    string = var.davinci_variable_ciam_passwordlessRequired_value
  }
}


resource "pingone_davinci_variable" "pingcli__ciam_recoveryLimit_company" {
  environment_id = var.pingone_environment_id
  name           = "ciam_recoveryLimit"
  context        = "company"
  data_type      = "number"
  mutable        = true
  min            = 0
  max            = 2000

  value = {
    string = var.davinci_variable_ciam_recoveryLimit_value
  }
}


resource "pingone_davinci_variable" "pingcli__ciam_recoveryValidationAttempts_flowInstance" {
  environment_id = var.pingone_environment_id
  name           = "ciam_recoveryValidationAttempts"
  context        = "flowInstance"
  data_type      = "number"
  mutable        = true
  min            = 0
  max            = 2000
}


resource "pingone_davinci_variable" "pingcli__ciam_requireMFA_company" {
  environment_id = var.pingone_environment_id
  name           = "ciam_requireMFA"
  context        = "company"
  data_type      = "boolean"
  mutable        = false
  display_name   = "Select whether to offer or require MFA"
  min            = 0
  max            = 2000

  value = {
    string = var.davinci_variable_ciam_requireMFA_value
  }
}


resource "pingone_davinci_variable" "pingcli__ciam_resendOtpAttempts_flowInstance" {
  environment_id = var.pingone_environment_id
  name           = "ciam_resendOtpAttempts"
  context        = "flowInstance"
  data_type      = "number"
  mutable        = true
  min            = 0
  max            = 2000
}


resource "pingone_davinci_variable" "pingcli__ciam_resendOtpLimit_company" {
  environment_id = var.pingone_environment_id
  name           = "ciam_resendOtpLimit"
  context        = "company"
  data_type      = "number"
  mutable        = true
  min            = 0
  max            = 2000

  value = {
    string = var.davinci_variable_ciam_resendOtpLimit_value
  }
}


resource "pingone_davinci_variable" "pingcli__ciam_sessionLengthInMinute_company" {
  environment_id = var.pingone_environment_id
  name           = "ciam_sessionLengthInMinute"
  context        = "company"
  data_type      = "number"
  mutable        = false
  min            = 0
  max            = 2000

  value = {
    string = var.davinci_variable_ciam_sessionLengthInMinute_value
  }
}


resource "pingone_davinci_variable" "pingcli__ciam_smsOtpEnabled_company" {
  environment_id = var.pingone_environment_id
  name           = "ciam_smsOtpEnabled"
  context        = "company"
  data_type      = "boolean"
  mutable        = true
  min            = 0
  max            = 2000

  value = {
    string = var.davinci_variable_ciam_smsOtpEnabled_value
  }
}


resource "pingone_davinci_variable" "pingcli__ciam_totpEnabled_company" {
  environment_id = var.pingone_environment_id
  name           = "ciam_totpEnabled"
  context        = "company"
  data_type      = "boolean"
  mutable        = false
  min            = 0
  max            = 2000

  value = {
    string = var.davinci_variable_ciam_totpEnabled_value
  }
}


resource "pingone_davinci_variable" "pingcli__ciam_verificationLimit_company" {
  environment_id = var.pingone_environment_id
  name           = "ciam_verificationLimit"
  context        = "company"
  data_type      = "number"
  mutable        = false
  min            = 0
  max            = 2000

  value = {
    string = var.davinci_variable_ciam_verificationLimit_value
  }
}


resource "pingone_davinci_variable" "pingcli__ciam_verificationValidationAttempts_flowInstance" {
  environment_id = var.pingone_environment_id
  name           = "ciam_verificationValidationAttempts"
  context        = "flowInstance"
  data_type      = "number"
  mutable        = true
  min            = 0
  max            = 2000
}


resource "pingone_davinci_variable" "pingcli__ciam_voiceOtpEnabled_company" {
  environment_id = var.pingone_environment_id
  name           = "ciam_voiceOtpEnabled"
  context        = "company"
  data_type      = "boolean"
  mutable        = true
  min            = 0
  max            = 2000

  value = {
    string = var.davinci_variable_ciam_voiceOtpEnabled_value
  }
}


resource "pingone_davinci_variable" "pingcli__companyContextVar_company" {
  environment_id = var.pingone_environment_id
  name           = "companyContextVar"
  context        = "company"
  data_type      = "string"
  mutable        = true
  min            = 0
  max            = 2000

  value = {
    string = var.davinci_variable_companyContextVar_value
  }
}


resource "pingone_davinci_variable" "pingcli__companyLogo_flowInstance" {
  environment_id = var.pingone_environment_id
  name           = "companyLogo"
  context        = "flowInstance"
  data_type      = "string"
  mutable        = true
  display_name   = "Url for company's logo image"
  min            = 0
  max            = 2000
}


resource "pingone_davinci_variable" "pingcli__companyName_flowInstance" {
  environment_id = var.pingone_environment_id
  name           = "companyName"
  context        = "flowInstance"
  data_type      = "string"
  mutable        = true
  min            = 0
  max            = 2000
}

resource "pingone_davinci_variable" "pingcli__FIDO2DisplayName_flowInstance" {
  environment_id = var.pingone_environment_id
  name           = "FIDO2DisplayName"
  context        = "flowInstance"
  data_type      = "string"
  mutable        = true
  min            = 0
  max            = 2000
}


resource "pingone_davinci_variable" "pingcli__flowRequireMFA_flowInstance" {
  environment_id = var.pingone_environment_id
  name           = "flowRequireMFA"
  context        = "flowInstance"
  data_type      = "boolean"
  mutable        = true
  min            = 0
  max            = 2000

  value = {
    string = var.davinci_variable_flowRequireMFA_value
  }
}


resource "pingone_davinci_variable" "pingcli__hideSkipButton_company" {
  environment_id = var.pingone_environment_id
  name           = "hideSkipButton"
  context        = "company"
  data_type      = "boolean"
  mutable        = true
  min            = 0
  max            = 2000

  value = {
    string = var.davinci_variable_hideSkipButton_value
  }
}


resource "pingone_davinci_variable" "pingcli__name_company" {
  environment_id = var.pingone_environment_id
  name           = "name"
  context        = "company"
  data_type      = "string"
  mutable        = true
  display_name   = "desc"
  min            = 0
  max            = 2000

  value = {
    string = var.davinci_variable_name_value
  }
}


resource "pingone_davinci_variable" "pingcli__registrationAgreementId_flowInstance" {
  environment_id = var.pingone_environment_id
  name           = "registrationAgreementId"
  context        = "flowInstance"
  data_type      = "string"
  mutable        = true
  min            = 0
  max            = 2000

  value = {
    string = var.davinci_variable_registrationAgreementId_value
  }
}


resource "pingone_davinci_variable" "pingcli__registrationPopulationId_flowInstance" {
  environment_id = var.pingone_environment_id
  name           = "registrationPopulationId"
  context        = "flowInstance"
  data_type      = "string"
  mutable        = true
  min            = 0
  max            = 2000

  value = {
    string = var.davinci_variable_registrationPopulationId_value
  }
}


resource "pingone_davinci_variable" "pingcli__testPW_flowInstance" {
  environment_id = var.pingone_environment_id
  name           = "testPW"
  context        = "flowInstance"
  data_type      = "string"
  mutable        = true
  min            = 0
  max            = 2000

  value = {
    string = var.davinci_variable_testPW_value
  }
}


resource "pingone_davinci_variable" "pingcli__username_flowInstance" {
  environment_id = var.pingone_environment_id
  name           = "username"
  context        = "flowInstance"
  data_type      = "string"
  mutable        = true
  min            = 0
  max            = 2000
}