# For protectWithSlaId assignments (using SLA domain UUID):
import {
  to = polaris_sla_domain_assignment.bronze
  id = "0e55e625-b78d-4e83-87f3-90313a980211"
}

# For doNotProtect assignments (using doNotProtect:<object_id1>,<object_id2>,...):
import {
  to = polaris_sla_domain_assignment.unprotected
  id = "doNotProtect:0e55e625-b78d-4e83-87f3-90313a980211,1a2b3c4d-5e6f-7890-abcd-ef1234567890"
}
