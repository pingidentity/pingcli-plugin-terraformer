
resource "pingone_davinci_connector_instance" "pingcli__Annotation" {
  environment_id = var.pingone_environment_id
  name           = "Annotation"

  connector = {
    id = "annotationConnector"
  }
}


resource "pingone_davinci_connector_instance" "pingcli__Challenge" {
  environment_id = var.pingone_environment_id
  name           = "Challenge"

  connector = {
    id = "challengeConnector"
  }
}


resource "pingone_davinci_connector_instance" "pingcli__Code-0020-Snippet" {
  environment_id = var.pingone_environment_id
  name           = "Code Snippet"

  connector = {
    id = "codeSnippetConnector"
  }
  properties = jsonencode({
    "code" : {
      "type" : "string",
      "value" : "${var.davinci_connection_Code_0020_Snippet_code}"
    },
    "inputSchema" : {
      "type" : "string",
      "value" : "${var.davinci_connection_Code_0020_Snippet_inputSchema}"
    },
    "outputSchema" : {
      "type" : "string",
      "value" : "${var.davinci_connection_Code_0020_Snippet_outputSchema}"
    }
  })
}


resource "pingone_davinci_connector_instance" "pingcli__DuplicateName" {
  environment_id = var.pingone_environment_id
  name           = "DuplicateName"

  connector = {
    id = "functionsConnector"
  }
}


resource "pingone_davinci_connector_instance" "pingcli__Error-0020-Message" {
  environment_id = var.pingone_environment_id
  name           = "Error Message"

  connector = {
    id = "errorConnector"
  }
}


resource "pingone_davinci_connector_instance" "pingcli__Flow-0020-Connector" {
  environment_id = var.pingone_environment_id
  name           = "Flow Connector"

  connector = {
    id = "flowConnector"
  }
}


resource "pingone_davinci_connector_instance" "pingcli__FormA-Test" {
  environment_id = var.pingone_environment_id
  name           = "FormA-Test"

  connector = {
    id = "pingOneFormsConnector"
  }
}


resource "pingone_davinci_connector_instance" "pingcli__Functions" {
  environment_id = var.pingone_environment_id
  name           = "Functions"

  connector = {
    id = "functionsConnector"
  }
}


resource "pingone_davinci_connector_instance" "pingcli__Http" {
  environment_id = var.pingone_environment_id
  name           = "Http"

  connector = {
    id = "httpConnector"
  }
}


resource "pingone_davinci_connector_instance" "pingcli__IDmission-0020---0020-OIDC-0020---0020-test" {
  environment_id = var.pingone_environment_id
  name           = "IDmission - OIDC - test"

  connector = {
    id = "idmissionOidcConnector"
  }
  properties = jsonencode({
    "customAuth" : {
      "type" : "json",
      "value" : jsonencode({
        "properties" : {
          "authTypeDropdown" : {
            "secure" : false,
            "value" : "${var.davinci_connection_IDmission_0020___0020_OIDC_0020___0020_test_customAuth_authTypeDropdown}"
          },
          "clientId" : {
            "secure" : false,
            "value" : "${var.davinci_connection_IDmission_0020___0020_OIDC_0020___0020_test_customAuth_clientId}"
          },
          "clientSecret" : {
            "secure" : true,
            "value" : "${var.davinci_connection_IDmission_0020___0020_OIDC_0020___0020_test_customAuth_clientSecret}"
          },
          "providerName" : {
            "secure" : false,
            "value" : "${var.davinci_connection_IDmission_0020___0020_OIDC_0020___0020_test_customAuth_providerName}"
          },
          "scope" : {
            "secure" : false,
            "value" : "${var.davinci_connection_IDmission_0020___0020_OIDC_0020___0020_test_customAuth_scope}"
          },
          "state" : {
            "secure" : false,
            "value" : "${var.davinci_connection_IDmission_0020___0020_OIDC_0020___0020_test_customAuth_state}"
          }
        }
      })
    }
  })
}


resource "pingone_davinci_connector_instance" "pingcli__IDmission-0020---0020-OIDC-0020---0020-test-0020-clone-0020-93b5" {
  environment_id = var.pingone_environment_id
  name           = "IDmission - OIDC - test clone 93b5"

  connector = {
    id = "idmissionOidcConnector"
  }
  properties = jsonencode({
    "customAuth" : {
      "type" : "json",
      "value" : jsonencode({
        "properties" : {
          "authTypeDropdown" : {
            "value" : "${var.davinci_connection_IDmission_0020___0020_OIDC_0020___0020_test_0020_clone_0020_93b5_customAuth_authTypeDropdown}"
          },
          "clientId" : {
            "value" : "${var.davinci_connection_IDmission_0020___0020_OIDC_0020___0020_test_0020_clone_0020_93b5_customAuth_clientId}"
          },
          "clientSecret" : {
            "secure" : true,
            "value" : "${var.davinci_connection_IDmission_0020___0020_OIDC_0020___0020_test_0020_clone_0020_93b5_customAuth_clientSecret}"
          },
          "providerName" : {
            "value" : "${var.davinci_connection_IDmission_0020___0020_OIDC_0020___0020_test_0020_clone_0020_93b5_customAuth_providerName}"
          },
          "scope" : {
            "value" : "${var.davinci_connection_IDmission_0020___0020_OIDC_0020___0020_test_0020_clone_0020_93b5_customAuth_scope}"
          },
          "state" : {
            "value" : "${var.davinci_connection_IDmission_0020___0020_OIDC_0020___0020_test_0020_clone_0020_93b5_customAuth_state}"
          }
        }
      })
    }
  })
}


resource "pingone_davinci_connector_instance" "pingcli__OIDC-0020--0026--0020-OAuth-0020-IdP-0020---0020-test" {
  environment_id = var.pingone_environment_id
  name           = "OIDC & OAuth IdP - test"

  connector = {
    id = "genericConnector"
  }
  properties = jsonencode({
    "customAuth" : {
      "type" : "json",
      "value" : jsonencode({
        "properties" : {
          "clientSecret" : {
            "secure" : true,
            "value" : "${var.davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020___0020_test_customAuth_clientSecret}"
          },
          "customAttributes" : {
            "value" : [
              {
                "attributeType" : "sk",
                "description" : "ID",
                "maxLength" : "300",
                "minLength" : "1",
                "name" : "id",
                "required" : true,
                "type" : "string",
                "value" : null
              },
              {
                "attributeType" : "sk",
                "description" : "Display Name",
                "maxLength" : "250",
                "minLength" : "1",
                "name" : "name",
                "required" : false,
                "type" : "string",
                "value" : null
              },
              {
                "attributeType" : "sk",
                "description" : "Email",
                "maxLength" : "250",
                "minLength" : "1",
                "name" : "email",
                "required" : false,
                "type" : "string",
                "value" : null
              }
            ]
          },
          "state" : {
            "value" : "${var.davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020___0020_test_customAuth_state}"
          }
        }
      })
    }
  })
}


resource "pingone_davinci_connector_instance" "pingcli__OIDC-0020--0026--0020-OAuth-0020-IdP-0020--testB" {
  environment_id = var.pingone_environment_id
  name           = "OIDC & OAuth IdP -testB"

  connector = {
    id = "genericConnector"
  }
  properties = jsonencode({
    "customAuth" : {
      "type" : "json",
      "value" : jsonencode({
        "properties" : {
          "authTypeDropdown" : {
            "value" : "${var.davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testB_customAuth_authTypeDropdown}"
          },
          "authorizationEndpoint" : {
            "value" : "${var.davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testB_customAuth_authorizationEndpoint}"
          },
          "clientId" : {
            "value" : "${var.davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testB_customAuth_clientId}"
          },
          "clientSecret" : {
            "secure" : true,
            "value" : "${var.davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testB_customAuth_clientSecret}"
          },
          "customAttributes" : {
            "value" : [
              {
                "attributeType" : "sk",
                "description" : "ID",
                "maxLength" : "300",
                "minLength" : "1",
                "name" : "id",
                "required" : true,
                "type" : "string",
                "value" : null
              },
              {
                "attributeType" : "sk",
                "description" : "Display Name",
                "maxLength" : "250",
                "minLength" : "1",
                "name" : "name",
                "required" : false,
                "type" : "string",
                "value" : null
              },
              {
                "attributeType" : "sk",
                "description" : "Email",
                "maxLength" : "250",
                "minLength" : "1",
                "name" : "email",
                "required" : false,
                "type" : "string",
                "value" : null
              }
            ]
          },
          "providerName" : {
            "value" : "${var.davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testB_customAuth_providerName}"
          },
          "scope" : {
            "value" : "${var.davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testB_customAuth_scope}"
          },
          "tokenEndpoint" : {
            "value" : "${var.davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testB_customAuth_tokenEndpoint}"
          },
          "userInfoEndpoint" : {
            "value" : [
              "https://auth.pingone.com/userInfo"
            ]
          }
        }
      })
    }
  })
}


resource "pingone_davinci_connector_instance" "pingcli__OIDC-0020--0026--0020-OAuth-0020-IdP-0020--testC" {
  environment_id = var.pingone_environment_id
  name           = "OIDC & OAuth IdP -testC"

  connector = {
    id = "genericConnector"
  }
  properties = jsonencode({
    "customAuth" : {
      "type" : "json",
      "value" : jsonencode({
        "properties" : {
          "authTypeDropdown" : {
            "value" : "${var.davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testC_customAuth_authTypeDropdown}"
          },
          "authorizationEndpoint" : {
            "value" : "${var.davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testC_customAuth_authorizationEndpoint}"
          },
          "clientId" : {
            "value" : "${var.davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testC_customAuth_clientId}"
          },
          "clientSecret" : {
            "secure" : true,
            "value" : "${var.davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testC_customAuth_clientSecret}"
          },
          "customAttributes" : {
            "value" : [
              {
                "attributeType" : "sk",
                "description" : "ID",
                "maxLength" : "300",
                "minLength" : "1",
                "name" : "id",
                "required" : true,
                "type" : "string",
                "value" : null
              },
              {
                "attributeType" : "sk",
                "description" : "Display Name",
                "maxLength" : "250",
                "minLength" : "1",
                "name" : "name",
                "required" : false,
                "type" : "string",
                "value" : null
              },
              {
                "attributeType" : "sk",
                "description" : "Email",
                "maxLength" : "250",
                "minLength" : "1",
                "name" : "email",
                "required" : false,
                "type" : "string",
                "value" : null
              }
            ]
          },
          "providerName" : {
            "value" : "${var.davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testC_customAuth_providerName}"
          },
          "scope" : {
            "value" : "${var.davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testC_customAuth_scope}"
          },
          "tokenEndpoint" : {
            "value" : "${var.davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testC_customAuth_tokenEndpoint}"
          },
          "userInfoEndpoint" : {
            "value" : [
              "https://auth.pingone.com/userInfo"
            ]
          }
        }
      })
    }
  })
}


resource "pingone_davinci_connector_instance" "pingcli__OIDC-0020--0026--0020-OAuth-0020-IdP-0020--testD" {
  environment_id = var.pingone_environment_id
  name           = "OIDC & OAuth IdP -testD"

  connector = {
    id = "genericConnector"
  }
  properties = jsonencode({
    "customAuth" : {
      "type" : "json",
      "value" : jsonencode({
        "properties" : {
          "authTypeDropdown" : {
            "value" : "${var.davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testD_customAuth_authTypeDropdown}"
          },
          "authorizationEndpoint" : {
            "value" : "${var.davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testD_customAuth_authorizationEndpoint}"
          },
          "clientId" : {
            "value" : "${var.davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testD_customAuth_clientId}"
          },
          "clientSecret" : {
            "secure" : true,
            "value" : "${var.davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testD_customAuth_clientSecret}"
          },
          "customAttributes" : {
            "value" : [
              {
                "attributeType" : "sk",
                "description" : "ID",
                "maxLength" : "300",
                "minLength" : "1",
                "name" : "id",
                "required" : true,
                "type" : "string",
                "value" : null
              },
              {
                "attributeType" : "sk",
                "description" : "Display Name",
                "maxLength" : "250",
                "minLength" : "1",
                "name" : "name",
                "required" : false,
                "type" : "string",
                "value" : null
              },
              {
                "attributeType" : "sk",
                "description" : "Email",
                "maxLength" : "250",
                "minLength" : "1",
                "name" : "email",
                "required" : false,
                "type" : "string",
                "value" : null
              }
            ]
          },
          "providerName" : {
            "value" : "${var.davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testD_customAuth_providerName}"
          },
          "scope" : {
            "value" : "${var.davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testD_customAuth_scope}"
          },
          "tokenEndpoint" : {
            "value" : "${var.davinci_connection_OIDC_0020__0026__0020_OAuth_0020_IdP_0020__testD_customAuth_tokenEndpoint}"
          },
          "userInfoEndpoint" : {
            "value" : [
              "https://auth.pingone.com/userInfo"
            ]
          }
        }
      })
    }
  })
}


resource "pingone_davinci_connector_instance" "pingcli__PingOne" {
  environment_id = var.pingone_environment_id
  name           = "PingOne"

  connector = {
    id = "pingOneSSOConnector"
  }
  properties = jsonencode({
    "clientId" : {
      "value" : "${var.davinci_connection_PingOne_clientId}"
    },
    "clientSecret" : {
      "value" : "${var.davinci_connection_PingOne_clientSecret}"
    },
    "envId" : {
      "value" : "${var.davinci_connection_PingOne_envId}"
    },
    "region" : {
      "value" : "${var.davinci_connection_PingOne_region}"
    }
  })
}


resource "pingone_davinci_connector_instance" "pingcli__PingOne-0020-Authentication" {
  environment_id = var.pingone_environment_id
  name           = "PingOne Authentication"

  connector = {
    id = "pingOneAuthenticationConnector"
  }
}


resource "pingone_davinci_connector_instance" "pingcli__PingOne-0020-MFA" {
  environment_id = var.pingone_environment_id
  name           = "PingOne MFA"

  connector = {
    id = "pingOneMfaConnector"
  }
  properties = jsonencode({
    "clientId" : {
      "value" : "${var.davinci_connection_PingOne_0020_MFA_clientId}"
    },
    "clientSecret" : {
      "value" : "${var.davinci_connection_PingOne_0020_MFA_clientSecret}"
    },
    "envId" : {
      "value" : "${var.davinci_connection_PingOne_0020_MFA_envId}"
    },
    "policyId" : {
      "value" : "${var.davinci_connection_PingOne_0020_MFA_policyId}"
    },
    "region" : {
      "value" : "${var.davinci_connection_PingOne_0020_MFA_region}"
    }
  })
}


resource "pingone_davinci_connector_instance" "pingcli__PingOne-0020-Notifications" {
  environment_id = var.pingone_environment_id
  name           = "PingOne Notifications"

  connector = {
    id = "notificationsConnector"
  }
  properties = jsonencode({
    "clientId" : {
      "value" : "${var.davinci_connection_PingOne_0020_Notifications_clientId}"
    },
    "clientSecret" : {
      "value" : "${var.davinci_connection_PingOne_0020_Notifications_clientSecret}"
    },
    "envId" : {
      "value" : "${var.davinci_connection_PingOne_0020_Notifications_envId}"
    },
    "region" : {
      "value" : "${var.davinci_connection_PingOne_0020_Notifications_region}"
    }
  })
}


resource "pingone_davinci_connector_instance" "pingcli__PingOne-0020-Protect" {
  environment_id = var.pingone_environment_id
  name           = "PingOne Protect"

  connector = {
    id = "pingOneRiskConnector"
  }
  properties = jsonencode({
    "clientId" : {
      "value" : "${var.davinci_connection_PingOne_0020_Protect_clientId}"
    },
    "clientSecret" : {
      "value" : "${var.davinci_connection_PingOne_0020_Protect_clientSecret}"
    },
    "envId" : {
      "value" : "${var.davinci_connection_PingOne_0020_Protect_envId}"
    },
    "region" : {
      "value" : "${var.davinci_connection_PingOne_0020_Protect_region}"
    }
  })
}


resource "pingone_davinci_connector_instance" "pingcli__String-0020-Manipulation" {
  environment_id = var.pingone_environment_id
  name           = "String Manipulation"

  connector = {
    id = "stringsConnector"
  }
}


resource "pingone_davinci_connector_instance" "pingcli__Teleport" {
  environment_id = var.pingone_environment_id
  name           = "Teleport"

  connector = {
    id = "nodeConnector"
  }
}


resource "pingone_davinci_connector_instance" "pingcli__Token-0020-Management" {
  environment_id = var.pingone_environment_id
  name           = "Token Management"

  connector = {
    id = "skOpenIdConnector"
  }
}


resource "pingone_davinci_connector_instance" "pingcli__User-0020-Policy" {
  environment_id = var.pingone_environment_id
  name           = "User Policy"

  connector = {
    id = "userPolicyConnector"
  }
}


resource "pingone_davinci_connector_instance" "pingcli__Variables" {
  environment_id = var.pingone_environment_id
  name           = "Variables"

  connector = {
    id = "variablesConnector"
  }
}