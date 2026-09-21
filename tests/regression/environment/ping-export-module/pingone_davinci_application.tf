
resource "pingone_davinci_application" "pingcli__DuplicateName" {
  environment_id = var.pingone_environment_id
  name           = "DuplicateName"

  api_key = {
    enabled = true
  }

  oauth = {
    grant_types = ["authorizationCode"]
    scopes      = ["openid", "profile"]
  }
}


resource "pingone_davinci_application" "pingcli__PingOne-0020-SSO-0020-Connection" {
  environment_id = var.pingone_environment_id
  name           = "PingOne SSO Connection"

  api_key = {
    enabled = true
  }

  oauth = {
    grant_types   = ["authorizationCode"]
    redirect_uris = ["https://auth.pingone.com/6b6bfb5d-8293-40d5-894e-0f74144334db/rp/callback/openid_connect"]
    scopes        = ["openid", "profile"]
  }
}


resource "pingone_davinci_application" "pingcli__testMultiFlowPolicy" {
  environment_id = var.pingone_environment_id
  name           = "testMultiFlowPolicy"

  api_key = {
    enabled = true
  }

  oauth = {
    grant_types = ["authorizationCode"]
    scopes      = ["openid", "profile"]
  }
}