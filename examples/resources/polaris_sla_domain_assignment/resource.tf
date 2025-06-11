data "polaris_sla_domain" "bronze" {
  name = "bronze"
}

# Create a tag rule for AWS EC2 instances.
resource "polaris_tag_rule" "aws_bronze" {
  name        = "aws-bronze"
  object_type = "AWS_EC2_INSTANCE"
  tag_key     = "backup"
  tag_value   = "true"
}

# Create a tag rule for Azure VM instances.
resource "polaris_tag_rule" "azure_bronze" {
  name        = "azure-bronze"
  object_type = "AZURE_VIRTUAL_MACHINE"
  tag_key     = "backup"
  tag_value   = "true"
}

# Assign the tag rules to the bronze SLA domain.
resource "polaris_sla_domain_assignment" "bronze" {
  sla_domain_id = data.polaris_sla_domain.bronze.id

  object_ids = [
    polaris_tag_rule.aws_bronze.id,
    polaris_tag_rule.azure_bronze.id,
  ]
}
