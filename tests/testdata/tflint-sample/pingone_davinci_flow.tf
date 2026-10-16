resource "pingone_davinci_flow" "pingcli__Login-0020-Flow" {
  environment_id = var.pingone_environment_id
  name           = "Login Flow"

  flow_status = {
    enabled = true
  }

  graph_data = {
    data = jsonencode({})
    renderer = jsonencode({
      "name" : "null"
    })
    elements = {
      nodes = {
        "0jvt7xvej2" = {
          data = {
            id              = "0jvt7xvej2"
            node_type       = "CONNECTION"
            connection_id   = pingone_davinci_connector_instance.pingcli__Code-0020-Snippet.id
            connector_id    = "codeSnippetConnector"
            name            = "Code Snippet"
            label           = "Code Snippet"
            capability_name = "code"
            type            = "action"
            status          = "configured"
            id_unique       = "0crkg5a1qc"
            properties = jsonencode({
              "nodeTitle" : {
                "value" : "Run Code Snippet"
              },
              "subFlowId" : {
                "value" : {
                  "label" : "Sub Flow",
                  "value" : pingone_davinci_flow.pingcli__Login-0020-Subflow.id
                }
              }
            })
          }
          position = {
            x = 1368.1972558640769
            y = 628.0249699048222
          }
          group      = "nodes"
          removed    = false
          selected   = false
          selectable = true
          locked     = false
          grabbable  = true
          pannable   = false
          classes    = ""
        }
      }
      edges = {}
    }
  }
}

resource "pingone_davinci_flow" "pingcli__Login-0020-Subflow" {
  environment_id = var.pingone_environment_id
  name           = "Login Subflow"

  flow_status = {
    enabled = true
  }

  graph_data = {
    data    = jsonencode({})
    renderer = jsonencode({
      "name" : "null"
    })
    elements = {
      nodes = {}
      edges = {}
    }
  }
}
