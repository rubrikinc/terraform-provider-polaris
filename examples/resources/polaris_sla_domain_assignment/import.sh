# For protectWithSlaId assignments (using SLA domain UUID):
% terraform import polaris_sla_domain_assignment.bronze 0e55e625-b78d-4e83-87f3-90313a980211

# For doNotProtect assignments (using doNotProtect:<object_id1>,<object_id2>,...):
% terraform import polaris_sla_domain_assignment.unprotected "doNotProtect:0e55e625-b78d-4e83-87f3-90313a980211"
