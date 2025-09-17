provider "kurrentcloud" {
  # optionally use ESC_TOKEN env var
  token = var.token

  # optionally use ESC_ORG_ID env var
  organization_id = var.organization_id

  # optionally use ESC_URL env var
  # you would normally not need to set it
  url = var.url

  # optionally use ESC_TOKEN_STORE env var
  # you would normally not need to set it
  token_store = var.token_store
}
