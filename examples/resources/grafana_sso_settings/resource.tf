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
    certificate_path          = "/certs/saml.crt"
    private_key_path          = "/certs/saml.key"
    idp_metadata_url          = "https://nexus.microsoftonline-p.com/federationmetadata/saml20/federationmetadata.xml"
    signature_algorithm       = "rsa-sha256"
    assertion_attribute_login = "login"
    assertion_attribute_email = "email"
    name_id_format            = "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"
  }
}

# Configure SSO using LDAP
resource "grafana_sso_settings" "ldap_sso_settings" {
  provider_name = "ldap"

  ldap_settings {
    enabled = "true"
    config {
      servers {
        host          = "127.0.0.1"
        port          = 389
        search_filter = "(cn=%s)"
        bind_dn       = "cn=admin,dc=grafana,dc=org"
        bind_password = "grafana"
        search_base_dns = [
          "dc=grafana,dc=org",
        ]
        attributes = {
          name      = "givenName"
          surname   = "sn"
          username  = "cn"
          member_of = "memberOf"
          email     = "email"
        }
        group_mappings {
          group_dn      = "cn=superadmins,dc=grafana,dc=org"
          org_role      = "Admin"
          org_id        = 1
          grafana_admin = true
        }
        group_mappings {
          group_dn = "cn=users,dc=grafana,dc=org"
          org_role = "Editor"
        }
        group_mappings {
          group_dn = "*"
          org_role = "Viewer"
        }
      }
    }
  }
}
