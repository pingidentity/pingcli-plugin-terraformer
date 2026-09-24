resource "pingone_davinci_connector_instance" "pingcli__Code-0020-Snippet" {
  environment_id = var.pingone_environment_id
  name           = "Code Snippet"

  connector = {
    id = "codeSnippetConnector"
  }
  properties = jsonencode({
    "code" : {
      "type" : "string",
      "value" : var.davinci_connection_Code_0020_Snippet_code
    },
    "inputSchema" : {
      "type" : "string",
      "value" : var.davinci_connection_Code_0020_Snippet_inputSchema
    }
  })
}

resource "pingone_davinci_connector_instance" "pingcli__Apple-0020-Login" {
  environment_id = var.pingone_environment_id
  name           = "Apple Login"

  connector = {
    id = "appleConnector"
  }
  properties = jsonencode({
    "customAuth" : {
      "type" : "json",
      "value" : jsonencode({
        "properties" : {
          "clientSecret" : {
            "secure" : true,
            "value" : var.davinci_connection_Apple_0020_Login_customAuth_clientSecret
          },
          "providerName" : {
            "value" : var.davinci_connection_Apple_0020_Login_customAuth_providerName
          }
        }
      })
    }
  })
}
