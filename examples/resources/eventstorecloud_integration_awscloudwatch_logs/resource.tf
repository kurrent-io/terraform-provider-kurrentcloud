variable "region" {
  type = string
}

variable "stage" {
  type = string
}

locals {
  describe_log_groups_arn = "arn:aws:logs:${var.region}:${data.aws_caller_identity.current.account_id}:log-group:*"
}

data "aws_caller_identity" "current" {}

resource "aws_cloudwatch_log_group" "esdb_logs" {
  name = "EventStoreLogs-${var.stage}"
}

resource "aws_iam_access_key" "esdb_logs" {
  user = aws_iam_user.esdb_logs.name
}

resource "aws_iam_user" "esdb_logs" {
  name = "esdb_logs_user-${var.stage}"
  path = "/esdb_logs/"
}

resource "aws_iam_user_policy" "esdb_logs" {
  name = "esdb_logs_user_policy-${var.stage}"
  user = aws_iam_user.esdb_logs.name

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": [
        "logs:DescribeLogGroups"        
      ],
      "Effect": "Allow",
      "Resource": "${local.describe_log_groups_arn}"
    },
    {
      "Action": [        
        "logs:CreateLogStream",
        "logs:DescribeLogStreams",
        "logs:PutLogEvents"
      ],
      "Effect": "Allow",
      "Resource": "${aws_cloudwatch_log_group.esdb_logs.arn}:*"
    }    
  ]
}
EOF
}

resource "kurrentcloud_integration_awscloudwatch_logs" "cloudwatch" {
  project_id        = var.project_id
  cluster_ids       = [var.cluster_id]
  description       = "send ESDB logs to AWS CloudWatch"
  access_key_id     = aws_iam_access_key.esdb_logs.id
  secret_access_key = aws_iam_access_key.esdb_logs.secret
  group_name        = aws_cloudwatch_log_group.esdb_logs.name
  region            = var.region
}
