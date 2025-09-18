variable "region" {
  description = "AWS region"
  type        = string
}

variable "stage" {
  description = "stage"
  type        = string
}

locals {
  metric_namespace = "esdb_metrics_${var.stage}"
}

resource "aws_iam_access_key" "esdb_metrics" {
  user = aws_iam_user.esdb_metrics.name
}

resource "aws_iam_user" "esdb_metrics" {
  name = "esdb_metrics_user-${var.stage}"
  path = "/esdb_metrics/"
}

resource "aws_iam_user_policy" "esdb_metrics" {
  name = "esdb_metrics_user_policy-${var.stage}"
  user = aws_iam_user.esdb_metrics.name

  # TODO: restrict to just the created group
  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": [
        "cloudwatch:PutMetricData"        
      ],
      "Effect": "Allow",
      "Resource": "*",
      "Condition": {
          "ForAnyValue:StringEqualsIgnoreCase": {
                "cloudwatch:namespace": [
                    "${local.metric_namespace}",
                    "${local.metric_namespace}/eventstoredb",
                    "${local.metric_namespace}/host"
                ]
            }
        }
    }        
  ]
}
EOF
}


resource "kurrentcloud_integration_awscloudwatch_metrics" "cloudwatch" {
  project_id        = var.project_id
  cluster_ids       = [var.cluster_id]
  description       = "send ESDB metrics to AWS CloudWatch"
  access_key_id     = aws_iam_access_key.esdb_metrics.id
  secret_access_key = aws_iam_access_key.esdb_metrics.secret
  region            = var.region
  namespace         = local.metric_namespace
}
