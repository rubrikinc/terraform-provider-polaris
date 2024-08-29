package provider

const (
	keyAccountID                                    = "account_id"
	keyActions                                      = "actions"
	keyAppID                                        = "app_id"
	keyAppName                                      = "app_name"
	keyAppSecret                                    = "app_secret"
	keyArchivalLocationID                           = "archival_location_id"
	keyARN                                          = "arn"
	keyAssumeRole                                   = "assume_role"
	keyBucketPrefix                                 = "bucket_prefix"
	keyBucketTags                                   = "bucket_tags"
	keyCredentials                                  = "credentials"
	keyCloud                                        = "cloud"
	keyCloudAccountID                               = "cloud_account_id"
	keyCloudNativeArchival                          = "cloud_native_archival"
	keyCloudNativeArchivalEncryption                = "cloud_native_archival_encryption"
	keyCloudNativeProtection                        = "cloud_native_protection"
	keyClusterName                                  = "cluster_name"
	keyClusterSecurityGroupID                       = "cluster_security_group_id"
	keyConnectionCommand                            = "connection_command"
	keyConnectionCommandExecuted                    = "connection_command_executed"
	keyConnectionStatus                             = "connection_status"
	keyContainerName                                = "container_name"
	keyCustomerManagedKey                           = "customer_managed_key"
	keyCustomerManagedPolicies                      = "customer_managed_policies"
	keyDataActions                                  = "data_actions"
	keyDeleteSnapshotsOnDestroy                     = "delete_snapshots_on_destroy"
	keyDescription                                  = "description"
	keyEC2RecoveryRolePath                          = "ec2_recovery_role_path"
	keyEmail                                        = "email"
	keyExocompute                                   = "exocompute"
	keyExocomputeID                                 = "exocompute_id"
	keyExternalID                                   = "external_id"
	keyFeature                                      = "feature"
	keyFeatures                                     = "features"
	keyFQDN                                         = "fqdn"
	keyHash                                         = "hash"
	keyHierarchy                                    = "hierarchy"
	keyHostAccountID                                = "host_account_id"
	keyHostCloudAccountID                           = "host_cloud_account_id"
	keyID                                           = "id"
	keyInstanceProfile                              = "instance_profile"
	keyInstanceProfileKeys                          = "instance_profile_keys"
	keyIPAddresses                                  = "ip_addresses"
	keyIsAccountOwner                               = "is_account_owner"
	keyIsOrgAdmin                                   = "is_org_admin"
	keyKey                                          = "key"
	keyKMSMasterKey                                 = "kms_master_key"
	keyLocationTemplate                             = "location_template"
	keyManagedPolicies                              = "managed_policies"
	keyManifest                                     = "manifest"
	keyName                                         = "name"
	keyNativeID                                     = "native_id"
	keyNodeSecurityGroupID                          = "node_security_group_id"
	keyNotActions                                   = "not_actions"
	keyNotDataActions                               = "not_data_actions"
	keyObjectIDs                                    = "object_ids"
	keyOperation                                    = "operation"
	keyPermissionGroups                             = "permission_groups"
	keyPermission                                   = "permission"
	keyPermissions                                  = "permissions"
	keyPermissionsHash                              = "permissions_hash"
	keyPodOverlayNetworkCIDR                        = "pod_overlay_network_cidr"
	keyPolarisAccount                               = "polaris_account"
	keyPolarisAWSAccount                            = "polaris_aws_account"
	keyPolarisAWSArchivalLocation                   = "polaris_aws_archival_location"
	keyPolarisAWSCNPAccount                         = "polaris_aws_cnp_account"
	keyPolarisAWSCNPAccountAttachments              = "polaris_aws_cnp_account_attachments"
	keyPolarisAWSCNPAccountTrustPolicy              = "polaris_aws_cnp_account_trust_policy"
	keyPolarisAWSCNPArtifacts                       = "polaris_aws_cnp_artifacts"
	keyPolarisAWSCNPPermissions                     = "polaris_aws_cnp_permissions"
	keyPolarisAWSExocompute                         = "polaris_aws_exocompute"
	keyPolarisAWSExocomputeClusterAttachment        = "polaris_aws_exocompute_cluster_attachment"
	keyPolarisAWSPrivateContainerRegistry           = "polaris_aws_private_container_registry"
	keyPolarisAzureArchivalLocation                 = "polaris_azure_archival_location"
	keyPolarisAzureExocompute                       = "polaris_azure_exocompute"
	keyPolarisAzurePermissions                      = "polaris_azure_permissions"
	keyPolarisAzureServicePrincipal                 = "polaris_azure_service_principal"
	keyPolarisAzureSubscription                     = "polaris_azure_subscription"
	keyPolarisCustomRole                            = "polaris_custom_role"
	keyPolarisDeployment                            = "polaris_deployment"
	keyPolarisFeatures                              = "polaris_features"
	keyPolarisManaged                               = "polaris_managed"
	keyPolarisRole                                  = "polaris_role"
	keyPolarisRoleAssignment                        = "polaris_role_assignment"
	keyPolarisRoleTemplate                          = "polaris_role_template"
	keyPolarisUser                                  = "polaris_user"
	keyPolicy                                       = "policy"
	keyProfile                                      = "profile"
	keyRedundancy                                   = "redundancy"
	keyRegion                                       = "region"
	keyRegions                                      = "regions"
	keyResourceGroupActions                         = "resource_group_actions"
	keyResourceGroupDataActions                     = "resource_group_data_actions"
	keyResourceGroupName                            = "resource_group_name"
	keyResourceGroupNotActions                      = "resource_group_not_actions"
	keyResourceGroupNotDataActions                  = "resource_group_not_data_actions"
	keyResourceGroupTags                            = "resource_group_tags"
	keyResourceGroupRegion                          = "resource_group_region"
	keyRole                                         = "role"
	keyRoleID                                       = "role_id"
	keyRoleIDs                                      = "role_ids"
	keyRoleKey                                      = "role_key"
	keyRoleKeys                                     = "role_keys"
	keySDKAuth                                      = "sdk_auth"
	keySnappableType                                = "snappable_type"
	keySQLDBProtection                              = "sql_db_protection"
	keySQLMIProtection                              = "sql_mi_protection"
	keySetupYAML                                    = "setup_yaml"
	keyStackARN                                     = "stack_arn"
	keyStatus                                       = "status"
	keyStorageAccountNamePrefix                     = "storage_account_name_prefix"
	keyStorageAccountRegion                         = "storage_account_region"
	keyStorageAccountTags                           = "storage_account_tags"
	keyStorageClass                                 = "storage_class"
	keyStorageTier                                  = "storage_tier"
	keySubnet                                       = "subnet"
	keySubnets                                      = "subnets"
	keySubscriptionActions                          = "subscription_actions"
	keySubscriptionDataActions                      = "subscription_data_actions"
	keySubscriptionID                               = "subscription_id"
	keySubscriptionName                             = "subscription_name"
	keySubscriptionNotActions                       = "subscription_not_actions"
	keySubscriptionNotDataActions                   = "subscription_not_data_actions"
	keyTenantDomain                                 = "tenant_domain"
	keyTenantID                                     = "tenant_id"
	keyTokenCache                                   = "token_cache"
	keyTokenCacheDir                                = "token_cache_dir"
	keyTokenCacheSecret                             = "token_cache_secret"
	keyTokenRefresh                                 = "token_refresh"
	keyUserAssignedManagedIdentityName              = "user_assigned_managed_identity_name"
	keyUserAssignedManagedIdentityPrincipalID       = "user_assigned_managed_identity_principal_id"
	keyUserAssignedManagedIdentityRegion            = "user_assigned_managed_identity_region"
	keyUserAssignedManagedIdentityResourceGroupName = "user_assigned_managed_identity_resource_group_name"
	keyUserEmail                                    = "user_email"
	keyURL                                          = "url"
	keyVaultName                                    = "vault_name"
	keyVersion                                      = "version"
	keyVPCID                                        = "vpc_id"
)
