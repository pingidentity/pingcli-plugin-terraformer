

resource "pingone_davinci_application_flow_policy" "pingcli__a" {
  environment_id         = var.pingone_environment_id
  davinci_application_id = pingone_davinci_application.pingcli__testMultiFlowPolicy.id
  name                   = "a"
  status                 = "enabled"

  flow_distributions = [
    {
      id      = pingone_davinci_flow.pingcli__OOTB-0020---0020-Device-0020-Management-0020---0020-Main-0020-Flow.id
      version = -1
    }
  ]
}


resource "pingone_davinci_application_flow_policy" "pingcli__b" {
  environment_id         = var.pingone_environment_id
  davinci_application_id = pingone_davinci_application.pingcli__testMultiFlowPolicy.id
  name                   = "b"
  status                 = "enabled"

  flow_distributions = [
    {
      id      = pingone_davinci_flow.pingcli__OOTB-0020---0020-Device-0020-Management-0020---0020-Main-0020-Flow.id
      version = -1
    }
  ]
}


resource "pingone_davinci_application_flow_policy" "pingcli__c" {
  environment_id         = var.pingone_environment_id
  davinci_application_id = pingone_davinci_application.pingcli__testMultiFlowPolicy.id
  name                   = "c"
  status                 = "enabled"

  flow_distributions = [
    {
      id      = pingone_davinci_flow.pingcli__OOTB-0020---0020-Password-0020-Reset-0020---0020-Main-0020-Flow.id
      version = -1
    }
  ]
}

resource "pingone_davinci_application_flow_policy" "pingcli__CIAM-0020-Plus-0020---0020-Registration-0020-and-0020-Authentication-0020-with-0020-Username-0020-and-0020-Password-0020---0020-Main-0020-Flow" {
  environment_id         = var.pingone_environment_id
  davinci_application_id = pingone_davinci_application.pingcli__PingOne-0020-SSO-0020-Connection.id
  name                   = "CIAM Plus - Registration and Authentication with Username and Password - Main Flow"
  status                 = "enabled"

  flow_distributions = [
    {
      id      = pingone_davinci_flow.pingcli__CIAM-0020-Plus-0020---0020-Registration-0020-and-0020-Authentication-0020-with-0020-Username-0020-and-0020-Password-0020---0020-Main-0020-Flow.id
      version = -1
      weight  = 100
    }
  ]
}


resource "pingone_davinci_application_flow_policy" "pingcli__CIAM-0020-Plus-0020---0020-Registration-0020-and-0020-Authentication-0020-with-0020-Username-0020-and-0020-Password-0020---0020-Main-0020-Flow_2" {
  environment_id         = var.pingone_environment_id
  davinci_application_id = pingone_davinci_application.pingcli__PingOne-0020-SSO-0020-Connection.id
  name                   = "CIAM Plus - Registration and Authentication with Username and Password - Main Flow"
  status                 = "enabled"

  flow_distributions = [
    {
      id      = pingone_davinci_flow.pingcli__CIAM-0020-Plus-0020---0020-Registration-0020-and-0020-Authentication-0020-with-0020-Username-0020-and-0020-Password-0020---0020-Main-0020-Flow.id
      version = -1
      weight  = 100
    }
  ]
}


resource "pingone_davinci_application_flow_policy" "pingcli__d" {
  environment_id         = var.pingone_environment_id
  davinci_application_id = pingone_davinci_application.pingcli__testMultiFlowPolicy.id
  name                   = "d"
  status                 = "enabled"

  trigger = {
    type = "AUTHENTICATION"
    configuration = {
      mfa = {
        enabled     = false
        time        = 0
        time_format = "min"
      }
      pwd = {
        enabled     = false
        time        = 0
        time_format = "min"
      }
    }
  }

  flow_distributions = [
    {
      id      = pingone_davinci_flow.pingcli__PingOne-0020-Session-0020-Main-0020-Flow.id
      version = -1
    }
  ]
}


resource "pingone_davinci_application_flow_policy" "pingcli__DuplicateName" {
  environment_id         = var.pingone_environment_id
  davinci_application_id = pingone_davinci_application.pingcli__DuplicateName.id
  name                   = "DuplicateName"
  status                 = "enabled"

  flow_distributions = [
    {
      id      = pingone_davinci_flow.pingcli__DuplicateName.id
      version = -1
    }
  ]
}


resource "pingone_davinci_application_flow_policy" "pingcli__e" {
  environment_id         = var.pingone_environment_id
  davinci_application_id = pingone_davinci_application.pingcli__testMultiFlowPolicy.id
  name                   = "e"
  status                 = "enabled"

  trigger = {
    type = "AUTHENTICATION"
    configuration = {
      mfa = {
        enabled     = false
        time        = 0
        time_format = "min"
      }
      pwd = {
        enabled     = false
        time        = 0
        time_format = "min"
      }
    }
  }

  flow_distributions = [
    {
      id      = pingone_davinci_flow.pingcli__PingOne-0020-Session-0020-Main-0020-Flow.id
      version = -1
    }
  ]
}


resource "pingone_davinci_application_flow_policy" "pingcli__f" {
  environment_id         = var.pingone_environment_id
  davinci_application_id = pingone_davinci_application.pingcli__testMultiFlowPolicy.id
  name                   = "f"
  status                 = "enabled"

  flow_distributions = [
    {
      id      = pingone_davinci_flow.pingcli__OOTB-0020---0020-Account-0020-Recovery-0020-by-0020-Email.id
      version = -1
    }
  ]
}


resource "pingone_davinci_application_flow_policy" "pingcli__g" {
  environment_id         = var.pingone_environment_id
  davinci_application_id = pingone_davinci_application.pingcli__testMultiFlowPolicy.id
  name                   = "g"
  status                 = "enabled"

  flow_distributions = [
    {
      id      = pingone_davinci_flow.pingcli__CIAM-0020-Plus-0020---0020-Account-0020-Registration-0020---0020-Subflow.id
      version = -1
    }
  ]
}


resource "pingone_davinci_application_flow_policy" "pingcli__h" {
  environment_id         = var.pingone_environment_id
  davinci_application_id = pingone_davinci_application.pingcli__testMultiFlowPolicy.id
  name                   = "h"
  status                 = "enabled"

  flow_distributions = [
    {
      id      = pingone_davinci_flow.pingcli__OOTB-0020---0020-Account-0020-Recovery-0020-by-0020-Email.id
      version = -1
    }
  ]
}


resource "pingone_davinci_application_flow_policy" "pingcli__OOTB-0020---0020-Account-0020-Recovery-0020-by-0020-Email" {
  environment_id         = var.pingone_environment_id
  davinci_application_id = pingone_davinci_application.pingcli__PingOne-0020-SSO-0020-Connection.id
  name                   = "OOTB - Account Recovery by Email"
  status                 = "enabled"

  flow_distributions = [
    {
      id      = pingone_davinci_flow.pingcli__OOTB-0020---0020-Account-0020-Recovery-0020-by-0020-Email.id
      version = -1
      weight  = 100
    }
  ]
}


resource "pingone_davinci_application_flow_policy" "pingcli__OOTB-0020---0020-Basic-0020-Profile-0020-Management" {
  environment_id         = var.pingone_environment_id
  davinci_application_id = pingone_davinci_application.pingcli__PingOne-0020-SSO-0020-Connection.id
  name                   = "OOTB - Basic Profile Management"
  status                 = "enabled"

  flow_distributions = [
    {
      id      = pingone_davinci_flow.pingcli__OOTB-0020---0020-Basic-0020-Profile-0020-Management.id
      version = -1
      weight  = 100
    }
  ]
}


resource "pingone_davinci_application_flow_policy" "pingcli__OOTB-0020---0020-Device-0020-Management-0020---0020-Main-0020-Flow" {
  environment_id         = var.pingone_environment_id
  davinci_application_id = pingone_davinci_application.pingcli__PingOne-0020-SSO-0020-Connection.id
  name                   = "OOTB - Device Management - Main Flow"
  status                 = "enabled"

  flow_distributions = [
    {
      id      = pingone_davinci_flow.pingcli__OOTB-0020---0020-Device-0020-Management-0020---0020-Main-0020-Flow.id
      version = -1
      weight  = 100
    }
  ]
}


resource "pingone_davinci_application_flow_policy" "pingcli__OOTB-0020---0020-Password-0020-Reset-0020---0020-Main-0020-Flow" {
  environment_id         = var.pingone_environment_id
  davinci_application_id = pingone_davinci_application.pingcli__PingOne-0020-SSO-0020-Connection.id
  name                   = "OOTB - Password Reset - Main Flow"
  status                 = "enabled"

  flow_distributions = [
    {
      id      = pingone_davinci_flow.pingcli__OOTB-0020---0020-Password-0020-Reset-0020---0020-Main-0020-Flow.id
      version = -1
      weight  = 100
    }
  ]
}


resource "pingone_davinci_application_flow_policy" "pingcli__OOTB-0020---0020-Passwordless-0020---0020-Registration-002C--0020-Authentication-002C--0020--0026--0020-Account-0020-Recovery-0020---0020-Main-0020-Flow" {
  environment_id         = var.pingone_environment_id
  davinci_application_id = pingone_davinci_application.pingcli__PingOne-0020-SSO-0020-Connection.id
  name                   = "OOTB - Passwordless - Registration, Authentication, & Account Recovery - Main Flow"
  status                 = "enabled"

  flow_distributions = [
    {
      id      = pingone_davinci_flow.pingcli__OOTB-0020---0020-Passwordless-0020---0020-Registration-002C--0020-Authentication-002C--0020--0026--0020-Account-0020-Recovery-0020---0020-Main-0020-Flow.id
      version = -1
      weight  = 100
    }
  ]
}


resource "pingone_davinci_application_flow_policy" "pingcli__OOTB-0020---0020-Passwordless-0020---0020-Registration-002C--0020-Authentication-002C--0020--0026--0020-Account-0020-Recovery-0020---0020-Main-0020-Flow_2" {
  environment_id         = var.pingone_environment_id
  davinci_application_id = pingone_davinci_application.pingcli__PingOne-0020-SSO-0020-Connection.id
  name                   = "OOTB - Passwordless - Registration, Authentication, & Account Recovery - Main Flow"
  status                 = "enabled"

  flow_distributions = [
    {
      id      = pingone_davinci_flow.pingcli__OOTB-0020---0020-Passwordless-0020---0020-Registration-002C--0020-Authentication-002C--0020--0026--0020-Account-0020-Recovery-0020---0020-Main-0020-Flow.id
      version = -1
      weight  = 100
    }
  ]
}


resource "pingone_davinci_application_flow_policy" "pingcli__OOTB-0020---0020-Passwordless-0020---0020-Registration-002C--0020-Authentication-002C--0020--0026--0020-Account-0020-Recovery-0020---0020-Main-0020-Flow_3" {
  environment_id         = var.pingone_environment_id
  davinci_application_id = pingone_davinci_application.pingcli__PingOne-0020-SSO-0020-Connection.id
  name                   = "OOTB - Passwordless - Registration, Authentication, & Account Recovery - Main Flow"
  status                 = "enabled"

  flow_distributions = [
    {
      id      = pingone_davinci_flow.pingcli__OOTB-0020---0020-Device-0020-Management-0020---0020-Main-0020-Flow.id
      version = -1
    }
  ]
}


resource "pingone_davinci_application_flow_policy" "pingcli__PingOne-0020---0020-Sign-0020-On-0020-and-0020-Registration" {
  environment_id         = var.pingone_environment_id
  davinci_application_id = pingone_davinci_application.pingcli__PingOne-0020-SSO-0020-Connection.id
  name                   = "PingOne - Sign On and Registration"
  status                 = "enabled"

  trigger = {
    type = "AUTHENTICATION"
  }

  flow_distributions = [
    {
      id      = pingone_davinci_flow.pingcli__PingOne-0020-Session-0020-Main-0020-Flow.id
      version = -1
      weight  = 100
    }
  ]
}