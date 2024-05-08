# Configure SSO for GitHub using OAuth2
resource "grafana_sso_settings" "github_sso_settings" {
  provider_name = "github"
  oauth2_settings {
    name                  = "Github"
    client_id             = "<your GitHub app client id>"
    client_secret         = "<your GitHub app client secret>"
    allow_sign_up         = true
    auto_login            = false
    scopes                = "user:email,read:org"
    team_ids              = "150,300"
    allowed_organizations = "[\"My Organization\", \"Octocats\"]"
    allowed_domains       = "mycompany.com mycompany.org"
  }
}

# Configure SSO using generic OAuth2
resource "grafana_sso_settings" "generic_sso_settings" {
  provider_name = "generic_oauth"
  oauth2_settings {
    name              = "Auth0"
    auth_url          = "https://<domain>/authorize"
    token_url         = "https://<domain>/oauth/token"
    api_url           = "https://<domain>/userinfo"
    client_id         = "<client id>"
    client_secret     = "<client secret>"
    allow_sign_up     = true
    auto_login        = false
    scopes            = "openid profile email offline_access"
    use_pkce          = true
    use_refresh_token = true
  }
}

# Configure SSO using SAML
resource "grafana_sso_settings" "saml_sso_settings" {
  provider_name = "saml"
  saml_settings {
    allow_sign_up             = true
    certificate_path          = "devenv/docker/blocks/auth/saml-enterprise/cert.crt"
    private_key_path          = "devenv/docker/blocks/auth/saml-enterprise/key.pem"
    idp_metadata_url          = "https://nexus.microsoftonline-p.com/federationmetadata/saml20/federationmetadata.xml"
    signature_algorithm       = "rsa-sha256"
    assertion_attribute_login = "login"
    assertion_attribute_email = "email"
    name_id_format            = "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"
  }
}
