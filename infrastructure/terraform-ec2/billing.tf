# A billing alarm is the cheapest insurance on a card-backed account: it
# costs nothing and turns "surprise bill" into "email on day three".
#
# The EstimatedCharges metric is only published to us-east-1, regardless of
# where anything actually runs, so this provider alias is required even when
# var.region is somewhere else.
provider "aws" {
  alias  = "billing"
  region = "us-east-1"
}

resource "aws_sns_topic" "billing" {
  provider = aws.billing
  name     = "${local.name}-billing-alerts"
}

# Subscription is only created if an address was supplied. AWS sends a
# confirmation email that has to be clicked before anything is delivered -
# an unconfirmed subscription silently drops alerts.
resource "aws_sns_topic_subscription" "billing_email" {
  provider  = aws.billing
  count     = var.alert_email != "" ? 1 : 0
  topic_arn = aws_sns_topic.billing.arn
  protocol  = "email"
  endpoint  = var.alert_email
}

resource "aws_cloudwatch_metric_alarm" "billing" {
  provider = aws.billing

  alarm_name          = "${local.name}-estimated-charges"
  alarm_description   = "Estimated AWS charges have exceeded $${var.billing_alarm_threshold_usd}. Free-tier usage should keep this at or near zero."
  comparison_operator = "GreaterThanThreshold"
  threshold           = var.billing_alarm_threshold_usd
  evaluation_periods  = 1

  namespace   = "AWS/Billing"
  metric_name = "EstimatedCharges"
  statistic   = "Maximum"
  # Billing metrics only update a few times a day, so anything shorter than
  # a six-hour period just evaluates the same stale value repeatedly.
  period = 21600

  dimensions = {
    Currency = "USD"
  }

  alarm_actions = [aws_sns_topic.billing.arn]

  # Missing data is normal early in a billing cycle before the first metric
  # is published; treating it as breaching would fire a false alarm on day one.
  treat_missing_data = "notBreaching"
}
