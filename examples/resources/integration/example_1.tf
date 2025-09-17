resource "kurrentcloud_integration" "opsgenie_issues" {
  project_id  = var.project_id
  description = "create OpsGenie alerts from issues"
  data = {
    sink    = "opsGenie"
    api_key = "<secret OpsGenie key here>"
    source  = "issues"
  }
}

resource "kurrentcloud_integration" "slack_notifications" {
  project_id  = var.project_id
  description = "send Slack a message when a notification happens"
  data = {
    sink       = "slack"
    token      = "<secret token here>"
    channel_id = "#esc-cluster-notifications"
    source     = "notifications"
  }
}
