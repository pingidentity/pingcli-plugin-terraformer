

resource "pingone_davinci_flow_deploy" "pingcli__-005B-cloned-005D--0020-1775670565498-0020---0020-DaV-Flow_1775670546" {
  environment_id = var.pingone_environment_id
  flow_id        = pingone_davinci_flow.pingcli__-005B-cloned-005D--0020-1775670565498-0020---0020-DaV-Flow_1775670546.id

  deploy_trigger_values = {
    deployed_version = pingone_davinci_flow.pingcli__-005B-cloned-005D--0020-1775670565498-0020---0020-DaV-Flow_1775670546.current_version
  }
}

resource "pingone_davinci_flow_deploy" "pingcli__CIAM-0020-Plus-0020---0020-Account-0020-Recovery-0020---0020-Email-0020---0020-Subflow" {
  environment_id = var.pingone_environment_id
  flow_id        = pingone_davinci_flow.pingcli__CIAM-0020-Plus-0020---0020-Account-0020-Recovery-0020---0020-Email-0020---0020-Subflow.id

  deploy_trigger_values = {
    deployed_version = pingone_davinci_flow.pingcli__CIAM-0020-Plus-0020---0020-Account-0020-Recovery-0020---0020-Email-0020---0020-Subflow.current_version
  }
}


resource "pingone_davinci_flow_deploy" "pingcli__CIAM-0020-Plus-0020---0020-Account-0020-Registration-0020---0020-Subflow" {
  environment_id = var.pingone_environment_id
  flow_id        = pingone_davinci_flow.pingcli__CIAM-0020-Plus-0020---0020-Account-0020-Registration-0020---0020-Subflow.id

  deploy_trigger_values = {
    deployed_version = pingone_davinci_flow.pingcli__CIAM-0020-Plus-0020---0020-Account-0020-Registration-0020---0020-Subflow.current_version
  }
}


resource "pingone_davinci_flow_deploy" "pingcli__CIAM-0020-Plus-0020---0020-Agreement-0020--0028-ToS-0029--0020---0020-Subflow" {
  environment_id = var.pingone_environment_id
  flow_id        = pingone_davinci_flow.pingcli__CIAM-0020-Plus-0020---0020-Agreement-0020--0028-ToS-0029--0020---0020-Subflow.id

  deploy_trigger_values = {
    deployed_version = pingone_davinci_flow.pingcli__CIAM-0020-Plus-0020---0020-Agreement-0020--0028-ToS-0029--0020---0020-Subflow.current_version
  }
}


resource "pingone_davinci_flow_deploy" "pingcli__CIAM-0020-Plus-0020---0020-Change-0020-Password-0020---0020-Subflow" {
  environment_id = var.pingone_environment_id
  flow_id        = pingone_davinci_flow.pingcli__CIAM-0020-Plus-0020---0020-Change-0020-Password-0020---0020-Subflow.id

  deploy_trigger_values = {
    deployed_version = pingone_davinci_flow.pingcli__CIAM-0020-Plus-0020---0020-Change-0020-Password-0020---0020-Subflow.current_version
  }
}


resource "pingone_davinci_flow_deploy" "pingcli__CIAM-0020-Plus-0020---0020-Device-0020-Authentication-0020---0020-Subflow" {
  environment_id = var.pingone_environment_id
  flow_id        = pingone_davinci_flow.pingcli__CIAM-0020-Plus-0020---0020-Device-0020-Authentication-0020---0020-Subflow.id

  deploy_trigger_values = {
    deployed_version = pingone_davinci_flow.pingcli__CIAM-0020-Plus-0020---0020-Device-0020-Authentication-0020---0020-Subflow.current_version
  }
}


resource "pingone_davinci_flow_deploy" "pingcli__CIAM-0020-Plus-0020---0020-Device-0020-Registration-0020---0020-Subflow" {
  environment_id = var.pingone_environment_id
  flow_id        = pingone_davinci_flow.pingcli__CIAM-0020-Plus-0020---0020-Device-0020-Registration-0020---0020-Subflow.id

  deploy_trigger_values = {
    deployed_version = pingone_davinci_flow.pingcli__CIAM-0020-Plus-0020---0020-Device-0020-Registration-0020---0020-Subflow.current_version
  }
}


resource "pingone_davinci_flow_deploy" "pingcli__CIAM-0020-Plus-0020---0020-Magic-0020-Link-0020-Authentication-0020---0020-Subflow" {
  environment_id = var.pingone_environment_id
  flow_id        = pingone_davinci_flow.pingcli__CIAM-0020-Plus-0020---0020-Magic-0020-Link-0020-Authentication-0020---0020-Subflow.id

  deploy_trigger_values = {
    deployed_version = pingone_davinci_flow.pingcli__CIAM-0020-Plus-0020---0020-Magic-0020-Link-0020-Authentication-0020---0020-Subflow.current_version
  }
}


resource "pingone_davinci_flow_deploy" "pingcli__CIAM-0020-Plus-0020---0020-Registration-0020-and-0020-Authentication-0020-with-0020-Username-0020-and-0020-Password-0020---0020-Main-0020-Flow" {
  environment_id = var.pingone_environment_id
  flow_id        = pingone_davinci_flow.pingcli__CIAM-0020-Plus-0020---0020-Registration-0020-and-0020-Authentication-0020-with-0020-Username-0020-and-0020-Password-0020---0020-Main-0020-Flow.id

  deploy_trigger_values = {
    deployed_version = pingone_davinci_flow.pingcli__CIAM-0020-Plus-0020---0020-Registration-0020-and-0020-Authentication-0020-with-0020-Username-0020-and-0020-Password-0020---0020-Main-0020-Flow.current_version
  }
}


resource "pingone_davinci_flow_deploy" "pingcli__CIAM-0020-Plus-0020---0020-SignOn-0020---0020-Subflow" {
  environment_id = var.pingone_environment_id
  flow_id        = pingone_davinci_flow.pingcli__CIAM-0020-Plus-0020---0020-SignOn-0020---0020-Subflow.id

  deploy_trigger_values = {
    deployed_version = pingone_davinci_flow.pingcli__CIAM-0020-Plus-0020---0020-SignOn-0020---0020-Subflow.current_version
  }
}


resource "pingone_davinci_flow_deploy" "pingcli__CIAM-0020-Plus-0020---0020-Verify-0020-Email-0020---0020-Subflow" {
  environment_id = var.pingone_environment_id
  flow_id        = pingone_davinci_flow.pingcli__CIAM-0020-Plus-0020---0020-Verify-0020-Email-0020---0020-Subflow.id

  deploy_trigger_values = {
    deployed_version = pingone_davinci_flow.pingcli__CIAM-0020-Plus-0020---0020-Verify-0020-Email-0020---0020-Subflow.current_version
  }
}


resource "pingone_davinci_flow_deploy" "pingcli__CIAM-0020-Plus-0020---0020-Verify-0020-Email-0020-MFA-0020---0020-Subflow" {
  environment_id = var.pingone_environment_id
  flow_id        = pingone_davinci_flow.pingcli__CIAM-0020-Plus-0020---0020-Verify-0020-Email-0020-MFA-0020---0020-Subflow.id

  deploy_trigger_values = {
    deployed_version = pingone_davinci_flow.pingcli__CIAM-0020-Plus-0020---0020-Verify-0020-Email-0020-MFA-0020---0020-Subflow.current_version
  }
}


resource "pingone_davinci_flow_deploy" "pingcli__DaV-Flow-postman" {
  environment_id = var.pingone_environment_id
  flow_id        = pingone_davinci_flow.pingcli__DaV-Flow-postman.id

  deploy_trigger_values = {
    deployed_version = pingone_davinci_flow.pingcli__DaV-Flow-postman.current_version
  }
}


resource "pingone_davinci_flow_deploy" "pingcli__DaV-Flow_1775670546" {
  environment_id = var.pingone_environment_id
  flow_id        = pingone_davinci_flow.pingcli__DaV-Flow_1775670546.id

  deploy_trigger_values = {
    deployed_version = pingone_davinci_flow.pingcli__DaV-Flow_1775670546.current_version
  }
}


resource "pingone_davinci_flow_deploy" "pingcli__DuplicateName" {
  environment_id = var.pingone_environment_id
  flow_id        = pingone_davinci_flow.pingcli__DuplicateName.id

  deploy_trigger_values = {
    deployed_version = pingone_davinci_flow.pingcli__DuplicateName.current_version
  }
}


resource "pingone_davinci_flow_deploy" "pingcli__flowContextVariable" {
  environment_id = var.pingone_environment_id
  flow_id        = pingone_davinci_flow.pingcli__flowContextVariable.id

  deploy_trigger_values = {
    deployed_version = pingone_davinci_flow.pingcli__flowContextVariable.current_version
  }
}


resource "pingone_davinci_flow_deploy" "pingcli__jsLinksTest" {
  environment_id = var.pingone_environment_id
  flow_id        = pingone_davinci_flow.pingcli__jsLinksTest.id

  deploy_trigger_values = {
    deployed_version = pingone_davinci_flow.pingcli__jsLinksTest.current_version
  }
}


resource "pingone_davinci_flow_deploy" "pingcli__multiselect-try" {
  environment_id = var.pingone_environment_id
  flow_id        = pingone_davinci_flow.pingcli__multiselect-try.id

  deploy_trigger_values = {
    deployed_version = pingone_davinci_flow.pingcli__multiselect-try.current_version
  }
}


resource "pingone_davinci_flow_deploy" "pingcli__OOTB-0020---0020-Account-0020-Recovery-0020---0020-Subflow" {
  environment_id = var.pingone_environment_id
  flow_id        = pingone_davinci_flow.pingcli__OOTB-0020---0020-Account-0020-Recovery-0020---0020-Subflow.id

  deploy_trigger_values = {
    deployed_version = pingone_davinci_flow.pingcli__OOTB-0020---0020-Account-0020-Recovery-0020---0020-Subflow.current_version
  }
}


resource "pingone_davinci_flow_deploy" "pingcli__OOTB-0020---0020-Account-0020-Recovery-0020-by-0020-Email" {
  environment_id = var.pingone_environment_id
  flow_id        = pingone_davinci_flow.pingcli__OOTB-0020---0020-Account-0020-Recovery-0020-by-0020-Email.id

  deploy_trigger_values = {
    deployed_version = pingone_davinci_flow.pingcli__OOTB-0020---0020-Account-0020-Recovery-0020-by-0020-Email.current_version
  }
}


resource "pingone_davinci_flow_deploy" "pingcli__OOTB-0020---0020-Account-0020-Registration-0020---0020-Subflow" {
  environment_id = var.pingone_environment_id
  flow_id        = pingone_davinci_flow.pingcli__OOTB-0020---0020-Account-0020-Registration-0020---0020-Subflow.id

  deploy_trigger_values = {
    deployed_version = pingone_davinci_flow.pingcli__OOTB-0020---0020-Account-0020-Registration-0020---0020-Subflow.current_version
  }
}


resource "pingone_davinci_flow_deploy" "pingcli__OOTB-0020---0020-Agreement-0020--0028-ToS-0029--0020---0020-Subflow" {
  environment_id = var.pingone_environment_id
  flow_id        = pingone_davinci_flow.pingcli__OOTB-0020---0020-Agreement-0020--0028-ToS-0029--0020---0020-Subflow.id

  deploy_trigger_values = {
    deployed_version = pingone_davinci_flow.pingcli__OOTB-0020---0020-Agreement-0020--0028-ToS-0029--0020---0020-Subflow.current_version
  }
}


resource "pingone_davinci_flow_deploy" "pingcli__OOTB-0020---0020-Basic-0020-Profile-0020-Management" {
  environment_id = var.pingone_environment_id
  flow_id        = pingone_davinci_flow.pingcli__OOTB-0020---0020-Basic-0020-Profile-0020-Management.id

  deploy_trigger_values = {
    deployed_version = pingone_davinci_flow.pingcli__OOTB-0020---0020-Basic-0020-Profile-0020-Management.current_version
  }
}


resource "pingone_davinci_flow_deploy" "pingcli__OOTB-0020---0020-Change-0020-Password-0020---0020-Subflow" {
  environment_id = var.pingone_environment_id
  flow_id        = pingone_davinci_flow.pingcli__OOTB-0020---0020-Change-0020-Password-0020---0020-Subflow.id

  deploy_trigger_values = {
    deployed_version = pingone_davinci_flow.pingcli__OOTB-0020---0020-Change-0020-Password-0020---0020-Subflow.current_version
  }
}


resource "pingone_davinci_flow_deploy" "pingcli__OOTB-0020---0020-Device-0020-Authentication-0020---0020-Subflow" {
  environment_id = var.pingone_environment_id
  flow_id        = pingone_davinci_flow.pingcli__OOTB-0020---0020-Device-0020-Authentication-0020---0020-Subflow.id

  deploy_trigger_values = {
    deployed_version = pingone_davinci_flow.pingcli__OOTB-0020---0020-Device-0020-Authentication-0020---0020-Subflow.current_version
  }
}


resource "pingone_davinci_flow_deploy" "pingcli__OOTB-0020---0020-Device-0020-Management-0020---0020-Main-0020-Flow" {
  environment_id = var.pingone_environment_id
  flow_id        = pingone_davinci_flow.pingcli__OOTB-0020---0020-Device-0020-Management-0020---0020-Main-0020-Flow.id

  deploy_trigger_values = {
    deployed_version = pingone_davinci_flow.pingcli__OOTB-0020---0020-Device-0020-Management-0020---0020-Main-0020-Flow.current_version
  }
}


resource "pingone_davinci_flow_deploy" "pingcli__OOTB-0020---0020-Device-0020-Registration-0020---0020-Subflow" {
  environment_id = var.pingone_environment_id
  flow_id        = pingone_davinci_flow.pingcli__OOTB-0020---0020-Device-0020-Registration-0020---0020-Subflow.id

  deploy_trigger_values = {
    deployed_version = pingone_davinci_flow.pingcli__OOTB-0020---0020-Device-0020-Registration-0020---0020-Subflow.current_version
  }
}


resource "pingone_davinci_flow_deploy" "pingcli__OOTB-0020---0020-Magic-0020-Link-0020-Authentication-0020---0020-Subflow" {
  environment_id = var.pingone_environment_id
  flow_id        = pingone_davinci_flow.pingcli__OOTB-0020---0020-Magic-0020-Link-0020-Authentication-0020---0020-Subflow.id

  deploy_trigger_values = {
    deployed_version = pingone_davinci_flow.pingcli__OOTB-0020---0020-Magic-0020-Link-0020-Authentication-0020---0020-Subflow.current_version
  }
}


resource "pingone_davinci_flow_deploy" "pingcli__OOTB-0020---0020-Password-0020-Reset-0020---0020-Main-0020-Flow" {
  environment_id = var.pingone_environment_id
  flow_id        = pingone_davinci_flow.pingcli__OOTB-0020---0020-Password-0020-Reset-0020---0020-Main-0020-Flow.id

  deploy_trigger_values = {
    deployed_version = pingone_davinci_flow.pingcli__OOTB-0020---0020-Password-0020-Reset-0020---0020-Main-0020-Flow.current_version
  }
}


resource "pingone_davinci_flow_deploy" "pingcli__OOTB-0020---0020-Passwordless-0020---0020-Registration-002C--0020-Authentication-002C--0020--0026--0020-Account-0020-Recovery-0020---0020-Main-0020-Flow" {
  environment_id = var.pingone_environment_id
  flow_id        = pingone_davinci_flow.pingcli__OOTB-0020---0020-Passwordless-0020---0020-Registration-002C--0020-Authentication-002C--0020--0026--0020-Account-0020-Recovery-0020---0020-Main-0020-Flow.id

  deploy_trigger_values = {
    deployed_version = pingone_davinci_flow.pingcli__OOTB-0020---0020-Passwordless-0020---0020-Registration-002C--0020-Authentication-002C--0020--0026--0020-Account-0020-Recovery-0020---0020-Main-0020-Flow.current_version
  }
}


resource "pingone_davinci_flow_deploy" "pingcli__OOTB-0020---0020-Verify-0020-Email-0020---0020-Subflow" {
  environment_id = var.pingone_environment_id
  flow_id        = pingone_davinci_flow.pingcli__OOTB-0020---0020-Verify-0020-Email-0020---0020-Subflow.id

  deploy_trigger_values = {
    deployed_version = pingone_davinci_flow.pingcli__OOTB-0020---0020-Verify-0020-Email-0020---0020-Subflow.current_version
  }
}


resource "pingone_davinci_flow_deploy" "pingcli__PingOne-0020-Session-0020-Main-0020-Flow" {
  environment_id = var.pingone_environment_id
  flow_id        = pingone_davinci_flow.pingcli__PingOne-0020-Session-0020-Main-0020-Flow.id

  deploy_trigger_values = {
    deployed_version = pingone_davinci_flow.pingcli__PingOne-0020-Session-0020-Main-0020-Flow.current_version
  }
}


resource "pingone_davinci_flow_deploy" "pingcli__PingOne-0020-Sign-0020-On-0020-with-0020-Registration-002C--0020-Password-0020-Reset-0020-and-0020-Recovery" {
  environment_id = var.pingone_environment_id
  flow_id        = pingone_davinci_flow.pingcli__PingOne-0020-Sign-0020-On-0020-with-0020-Registration-002C--0020-Password-0020-Reset-0020-and-0020-Recovery.id

  deploy_trigger_values = {
    deployed_version = pingone_davinci_flow.pingcli__PingOne-0020-Sign-0020-On-0020-with-0020-Registration-002C--0020-Password-0020-Reset-0020-and-0020-Recovery.current_version
  }
}


resource "pingone_davinci_flow_deploy" "pingcli__pingoneFormsTest" {
  environment_id = var.pingone_environment_id
  flow_id        = pingone_davinci_flow.pingcli__pingoneFormsTest.id

  deploy_trigger_values = {
    deployed_version = pingone_davinci_flow.pingcli__pingoneFormsTest.current_version
  }
}


resource "pingone_davinci_flow_deploy" "pingcli__testCodeSnippetConnector" {
  environment_id = var.pingone_environment_id
  flow_id        = pingone_davinci_flow.pingcli__testCodeSnippetConnector.id

  deploy_trigger_values = {
    deployed_version = pingone_davinci_flow.pingcli__testCodeSnippetConnector.current_version
  }
}