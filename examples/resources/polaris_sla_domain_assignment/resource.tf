data "polaris_sla_domain" "bronze" {
  name = "bronze"
}

# Create a tag rule for AWS EC2 instances.
resource "polaris_tag_rule" "aws_bronze" {
  name        = "aws-bronze"
  object_type = "AWS_EC2_INSTANCE"

  tag {
    key    = "backup"
    values = ["true"]
  }
}

# Create a tag rule for Azure VM instances.
resource "polaris_tag_rule" "azure_bronze" {
  name        = "azure-bronze"
  object_type = "AZURE_VIRTUAL_MACHINE"

  tag {
    key    = "backup"
    values = ["true"]
  }
}

# Assign the tag rules to the bronze SLA domain.
resource "polaris_sla_domain_assignment" "bronze" {
  sla_domain_id = data.polaris_sla_domain.bronze.id

  object_ids = [
    polaris_tag_rule.aws_bronze.id,
    polaris_tag_rule.azure_bronze.id,
  ]
}

# Create a tag rule for development instances that should not be protected.
resource "polaris_tag_rule" "dev_instances" {
  name        = "dev-instances"
  object_type = "AWS_EC2_INSTANCE"

  tag {
    key    = "environment"
    values = ["dev"]
  }
}

# Mark development instances as Do Not Protect.
resource "polaris_sla_domain_assignment" "unprotected" {
  assignment_type             = "doNotProtect"
  existing_snapshot_retention = "RETAIN_SNAPSHOTS"

  object_ids = [
    polaris_tag_rule.dev_instances.id,
  ]
}
