// Copyright 2026 Rubrik, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
// DEALINGS IN THE SOFTWARE.

package provider

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/azure"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/devops"
	gqlazure "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/azure"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/hierarchy"
)

const dataSourceObjectDescription = `
The ´polaris_object´ data source is used to look up an RSC hierarchy object by
name and type. This is useful for finding the ID of an object when only its
name and type are known.

Supported object types:
  * ´AwsNativeAccount´ - AWS Native Account
  * ´AwsNativeEbsVolume´ - AWS Native EBS Volume
  * ´AwsNativeEc2Instance´ - AWS Native EC2 Instance
  * ´AwsNativeRdsInstance´ - AWS Native RDS Instance
  * ´AzureDevOpsOrganization´ - Azure DevOps Organization
  * ´AzureDevOpsProject´ - Azure DevOps Project
  * ´AzureDevOpsRepository´ - Azure DevOps Repository
  * ´AzureNativeResourceGroup´ - Azure Native Resource Group
  * ´AzureNativeSubscription´ - Azure Native Subscription
  * ´AzureNativeVirtualMachine´ - Azure Native Virtual Machine
  * ´AzureSqlManagedInstanceServer´ - Azure SQL Managed Instance Server
  * ´CloudNativeTagRule´ - Cloud Native Tag Rule
  * ´GitHubOrganization´ - GitHub Organization
  * ´GitHubRepository´ - GitHub Repository

-> **Note:** Azure resource group and SQL Managed Instance server names are
only unique within a subscription. When a name is shared across
subscriptions, set ´subscription_id´ to the parent subscription's RSC cloud
account ID to disambiguate; otherwise the lookup returns a "multiple objects
found" error.

-> **Note:** Azure DevOps project and repository names, and GitHub repository
names, are only unique within their parent. When a name is shared across
parents, set ´org_id´ (for ´AzureDevOpsProject´ or ´GitHubRepository´)
or ´org_id´ and/or ´project_id´ (for ´AzureDevOpsRepository´) to
disambiguate; otherwise the lookup returns a "multiple objects found" error.
`

var (
	_ datasource.DataSource                   = &objectDataSource{}
	_ datasource.DataSourceWithConfigure      = &objectDataSource{}
	_ datasource.DataSourceWithValidateConfig = &objectDataSource{}
)

type objectDataSource struct {
	client *client
}

type objectModel struct {
	ID             types.String   `tfsdk:"id"`
	Name           types.String   `tfsdk:"name"`
	ObjectType     types.String   `tfsdk:"object_type"`
	SubscriptionID types.String   `tfsdk:"subscription_id"`
	OrgID          types.String   `tfsdk:"org_id"`
	ProjectID      types.String   `tfsdk:"project_id"`
	Timeouts       timeouts.Value `tfsdk:"timeouts"`
}

func newObjectDataSource() datasource.DataSource {
	return &objectDataSource{}
}

func (d *objectDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, res *datasource.MetadataResponse) {
	tflog.Trace(ctx, "objectDataSource.Metadata")

	res.TypeName = req.ProviderTypeName + "_object"
}

func (d *objectDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, res *datasource.SchemaResponse) {
	tflog.Trace(ctx, "objectDataSource.Schema")

	// objectTypes is the set of object types the polaris_object data source
	// can look up. Used both for input validation and in the object_type
	// description.
	var objectTypes = []string{
		"AwsNativeAccount",
		"AwsNativeEbsVolume",
		"AwsNativeEc2Instance",
		"AwsNativeRdsInstance",
		"AzureDevOpsOrganization",
		"AzureDevOpsProject",
		"AzureDevOpsRepository",
		"AzureNativeResourceGroup",
		"AzureNativeSubscription",
		"AzureNativeVirtualMachine",
		"AzureSqlManagedInstanceServer",
		"CloudNativeTagRule",
		"GitHubOrganization",
		"GitHubRepository",
	}

	res.Schema = schema.Schema{
		Description: description(dataSourceObjectDescription),
		Attributes: map[string]schema.Attribute{
			keyID: schema.StringAttribute{
				Computed:    true,
				Description: "Object ID (UUID).",
			},
			keyName: schema.StringAttribute{
				Required:    true,
				Description: "Exact object name to search for.",
				Validators: []validator.String{
					isNotWhiteSpace(),
				},
			},
			keyObjectType: schema.StringAttribute{
				Required:    true,
				Description: fmt.Sprintf("Object type. %s.", possibleValues(objectTypes)),
				Validators: []validator.String{
					stringvalidator.OneOf(objectTypes...),
				},
			},
			keySubscriptionID: schema.StringAttribute{
				Optional: true,
				Description: "RSC cloud account ID of the parent Azure subscription (UUID). May be set when " +
					"`object_type` is `AzureNativeResourceGroup` or `AzureSqlManagedInstanceServer` to " +
					"disambiguate a name shared across subscriptions; must not be set for other object types.",
				Validators: []validator.String{
					isUUID(),
				},
			},
			keyOrgID: schema.StringAttribute{
				Optional: true,
				Description: "RSC object ID of the parent organization (UUID). May be set when `object_type` is " +
					"`AzureDevOpsProject` to disambiguate a project name shared across organizations, when " +
					"`object_type` is `AzureDevOpsRepository` to disambiguate a repository name shared across " +
					"projects, or when `object_type` is `GitHubRepository` to disambiguate a repository name " +
					"shared across organizations; must not be set for other object types.",
				Validators: []validator.String{
					isUUID(),
				},
			},
			keyProjectID: schema.StringAttribute{
				Optional: true,
				Description: "RSC object ID of the parent Azure DevOps project (UUID). May be set when `object_type` " +
					"is `AzureDevOpsRepository` to disambiguate a repository name shared across projects; must not " +
					"be set for other object types.",
				Validators: []validator.String{
					isUUID(),
				},
			},
			// The read timeout is used by the AwsNativeAccount retry loop which
			// polls the hierarchy until an active account appears. Other object
			// types return immediately and are unaffected by this timeout.
			keyTimeouts: timeouts.AttributesWithOpts(ctx, timeouts.Opts{
				ReadDescription: "How long to wait for the object to appear in the hierarchy. Default is `5m`.",
			}),
		},
	}
}

func (d *objectDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, res *datasource.ConfigureResponse) {
	tflog.Trace(ctx, "objectDataSource.Configure")

	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*client)
}

func (d *objectDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, res *datasource.ValidateConfigResponse) {
	tflog.Trace(ctx, "objectDataSource.ValidateConfig")

	var config objectModel
	res.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if res.Diagnostics.HasError() {
		return
	}

	res.Diagnostics.Append(validateObjectConfig(config)...)
}

// validateObjectConfig holds the plan-time, client-free validation rules for
// the data source so they can be unit-tested in isolation. It enforces that the
// optional parent-ID fields are only set for the object types they apply to.
func validateObjectConfig(config objectModel) diag.Diagnostics {
	var diags diag.Diagnostics

	// object_type drives which parent-ID fields are allowed. When it is not
	// known yet, for example because it references another resource, there is
	// nothing to check.
	if config.ObjectType.IsNull() || config.ObjectType.IsUnknown() {
		return diags
	}

	// Each parent-ID field may only be set for the listed object types.
	constraints := []struct {
		key     string
		value   types.String
		allowed []string
	}{{
		key:     keySubscriptionID,
		value:   config.SubscriptionID,
		allowed: []string{"AzureNativeResourceGroup", "AzureSqlManagedInstanceServer"},
	}, {
		key:     keyOrgID,
		value:   config.OrgID,
		allowed: []string{"AzureDevOpsProject", "AzureDevOpsRepository", "GitHubRepository"},
	}, {
		key:     keyProjectID,
		value:   config.ProjectID,
		allowed: []string{"AzureDevOpsRepository"},
	}}

	objectType := config.ObjectType.ValueString()
	for _, c := range constraints {
		if c.value.IsNull() || c.value.IsUnknown() {
			continue
		}
		if slices.Contains(c.allowed, objectType) {
			continue
		}

		diags.AddAttributeError(path.Root(c.key), fmt.Sprintf("Invalid %s", c.key),
			fmt.Sprintf("%s may only be set when %s is one of: %s.", c.key, keyObjectType, strings.Join(c.allowed, ", ")))
	}

	return diags
}

func (d *objectDataSource) Read(ctx context.Context, req datasource.ReadRequest, res *datasource.ReadResponse) {
	tflog.Trace(ctx, "objectDataSource.Read")

	var config objectModel
	res.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if res.Diagnostics.HasError() {
		return
	}

	readTimeout, diags := config.Timeouts.Read(ctx, 5*time.Minute)
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	polarisClient, err := d.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	name := config.Name.ValueString()
	objectType := hierarchy.ObjectType(config.ObjectType.ValueString())

	api := hierarchy.Wrap(polarisClient.GQL)

	// Filters for workload-level object types. Unlike container-level types
	// (e.g. AwsNativeAccount, AzureNativeSubscription), workload objects do not
	// carry RSC feature-status metadata, so activity is determined via these
	// server-side filters rather than inspecting the returned feature list.
	activeFilters := activeObjectFilters()

	var objects []hierarchy.Object
	switch objectType {
	case "AwsNativeAccount":
		// Container-level type: the API can return multiple entries for the
		// same account name (e.g. an account added to RSC more than once).
		// Activity is determined by inspecting the RSC feature status on each
		// result rather than using server-side filters.
		//
		// A newly onboarded AWS account may not appear in the hierarchy
		// immediately after creation because the polaris_aws_cnp_account
		// resource only registers the account while the hierarchy object is
		// created asynchronously after the polaris_aws_cnp_account_attachments
		// resource finalizes the account setup. When polaris_object depends on
		// the account, it can run before the hierarchy has caught up. We retry
		// until an active account is found or the read timeout is reached.
		for {
			results, err := hierarchy.ObjectsByName[hierarchy.AWSNativeAccount](ctx, api, name, hierarchy.WorkloadAllSubHierarchyType)
			if err != nil {
				res.Diagnostics.AddError("Failed to look up objects", err.Error())
				return
			}

			for _, r := range results {
				var active bool
				for _, feature := range r.Features {
					switch feature.Status {
					case hierarchy.StatusAdded, hierarchy.StatusRefreshed, hierarchy.StatusRefreshing:
						active = true
					default:
						tflog.Debug(ctx, "skipping account because it is not active", map[string]any{
							"account": r.Object.Name,
							"status":  feature.Status,
						})
					}
					if active {
						objects = append(objects, r.Object)
						break
					}
				}
			}
			if len(objects) > 0 {
				break
			}

			tflog.Debug(ctx, "no active account found in hierarchy, retrying", map[string]any{
				"name": name,
			})

			select {
			case <-ctx.Done():
				res.Diagnostics.AddError("Timed out waiting for object", fmt.Sprintf(
					"timed out waiting for active object with name %q and type %q: %d result(s) returned, none active",
					name, objectType, len(results)))
				return
			case <-time.After(10 * time.Second):
			}
		}
	case "AwsNativeEbsVolume":
		results, err := hierarchy.ObjectsByName[hierarchy.AWSNativeEBSVolume](ctx, api, name, hierarchy.WorkloadAllSubHierarchyType, activeFilters...)
		if err != nil {
			res.Diagnostics.AddError("Failed to look up objects", err.Error())
			return
		}

		for _, r := range results {
			objects = append(objects, r.Object)
		}
	case "AwsNativeEc2Instance":
		results, err := hierarchy.ObjectsByName[hierarchy.AWSNativeEC2Instance](ctx, api, name, hierarchy.WorkloadAllSubHierarchyType, activeFilters...)
		if err != nil {
			res.Diagnostics.AddError("Failed to look up objects", err.Error())
			return
		}

		for _, r := range results {
			objects = append(objects, r.Object)
		}
	case "AwsNativeRdsInstance":
		results, err := hierarchy.ObjectsByName[hierarchy.AWSNativeRDSInstance](ctx, api, name, hierarchy.WorkloadAllSubHierarchyType, activeFilters...)
		if err != nil {
			res.Diagnostics.AddError("Failed to look up objects", err.Error())
			return
		}

		for _, r := range results {
			objects = append(objects, r.Object)
		}
	case "AzureDevOpsOrganization":
		orgs, err := devops.Wrap(polarisClient).AzureOrganizationsByName(ctx, name,
			activeObjectFilters(hierarchy.Filter{Field: "NAME_EXACT_MATCH", Texts: []string{name}})...)
		if err != nil {
			res.Diagnostics.AddError("Failed to look up Azure DevOps organizations", err.Error())
			return
		}

		for _, org := range orgs {
			objects = append(objects, hierarchy.Object{
				ID:         org.ID,
				Name:       org.Name,
				ObjectType: hierarchy.ObjectType(org.ObjectType),
			})
		}
	case "AzureDevOpsProject":
		// The inventory query returns a 500 error for the Azure DevOps project
		// type, so route all Azure DevOps hierarchy lookups through the
		// dedicated queries instead.
		projects, err := devops.Wrap(polarisClient).AzureProjectsByName(ctx, name,
			activeObjectFilters(hierarchy.Filter{Field: "NAME_EXACT_MATCH", Texts: []string{name}})...)
		if err != nil {
			res.Diagnostics.AddError("Failed to look up Azure DevOps projects", err.Error())
			return
		}

		// Project names are only unique within an organization, so when
		// org_id is set, narrow to that organization.
		orgID := config.OrgID.ValueString()
		for _, project := range projects {
			if orgID != "" && project.OrgID.String() != orgID {
				continue
			}
			objects = append(objects, hierarchy.Object{
				ID:         project.ID,
				Name:       project.Name,
				ObjectType: hierarchy.ObjectType(project.ObjectType),
			})
		}
	case "AzureDevOpsRepository":
		repos, err := devops.Wrap(polarisClient).AzureRepositoriesByName(ctx, name,
			activeObjectFilters(hierarchy.Filter{Field: "NAME_EXACT_MATCH", Texts: []string{name}})...)
		if err != nil {
			res.Diagnostics.AddError("Failed to look up Azure DevOps repositories", err.Error())
			return
		}

		// Repository names are only unique within a project, so when
		// org_id and/or project_id are set, narrow to that
		// organization and project.
		orgID := config.OrgID.ValueString()
		projectID := config.ProjectID.ValueString()
		for _, repo := range repos {
			if orgID != "" && repo.OrgID.String() != orgID {
				continue
			}
			if projectID != "" && repo.ProjectID.String() != projectID {
				continue
			}
			objects = append(objects, hierarchy.Object{
				ID:         repo.ID,
				Name:       repo.Name,
				ObjectType: hierarchy.ObjectType(repo.ObjectType),
			})
		}
	case "AzureNativeResourceGroup":
		// Azure resource groups are exposed in the inventory by two different
		// types, AzureNativeResourceGroupBase and AzureNativeResourceGroup.
		// The first is returned by generic inventory queries, the second
		// contains the data we needs so route the resource group lookup through
		// the dedicated query instead.
		filters := gqlazure.ResourceGroupFilters{
			NameSubstring: name,
		}
		if subscriptionID := config.SubscriptionID.ValueString(); subscriptionID != "" {
			id, err := uuid.Parse(subscriptionID)
			if err != nil {
				res.Diagnostics.AddError("Invalid subscription ID", err.Error())
				return
			}
			nativeSub, err := azure.Wrap(polarisClient).NativeSubscriptionByCloudAccountID(ctx, id)
			if err != nil {
				res.Diagnostics.AddError("Failed to lookup subscription", err.Error())
				return
			}
			filters.SubscriptionIDs = append(filters.SubscriptionIDs, nativeSub.ID)
		}

		groups, err := azure.Wrap(polarisClient).NativeResourceGroupsByFilter(ctx, filters)
		if err != nil {
			res.Diagnostics.AddError("Failed to read Azure resource groups", err.Error())
			return
		}
		for _, group := range groups {
			if group.Name != name {
				continue
			}

			id, err := uuid.Parse(group.ID)
			if err != nil {
				res.Diagnostics.AddError("Invalid resource group ID", err.Error())
				return
			}
			objects = append(objects, hierarchy.Object{
				ID:         id,
				Name:       group.Name,
				ObjectType: "AzureNativeResourceGroup",
			})
		}
	case "AzureNativeSubscription":
		// Container-level type: same feature-status strategy as AwsNativeAccount.
		results, err := hierarchy.ObjectsByName[hierarchy.AzureNativeSubscription](ctx, api, name, hierarchy.WorkloadAllSubHierarchyType)
		if err != nil {
			res.Diagnostics.AddError("Failed to look up objects", err.Error())
			return
		}

		for _, r := range results {
			var active bool
			for _, feature := range r.Features {
				switch feature.Status {
				case hierarchy.StatusAdded, hierarchy.StatusRefreshed, hierarchy.StatusRefreshing:
					active = true
				default:
					tflog.Debug(ctx, "skipping subscription because it is not active", map[string]any{
						"subscription": r.Object.Name,
						"status":       feature.Status,
					})
				}
				if active {
					objects = append(objects, r.Object)
					break
				}
			}
		}
	case "AzureNativeVirtualMachine":
		results, err := hierarchy.ObjectsByName[hierarchy.AzureNativeVirtualMachine](ctx, api, name, hierarchy.WorkloadAzureVM, activeFilters...)
		if err != nil {
			res.Diagnostics.AddError("Failed to look up objects", err.Error())
			return
		}

		for _, r := range results {
			objects = append(objects, r.Object)
		}
	case "AzureSqlManagedInstanceServer":
		results, err := hierarchy.ObjectsByName[hierarchy.AzureSQLManagedInstanceServer](ctx, api, name, hierarchy.WorkloadAllSubHierarchyType, activeFilters...)
		if err != nil {
			res.Diagnostics.AddError("Failed to look up objects", err.Error())
			return
		}

		// SQL Managed Instance server names are only unique within a
		// subscription, and the inventory query has no subscription filter, so
		// when subscription_id is set, narrow to that subscription client-side.
		// The node's accountConnectionId is the RSC cloud account ID, so no
		// native FID translation is needed.
		subscriptionID := config.SubscriptionID.ValueString()
		for _, r := range results {
			if subscriptionID != "" && r.ResourceGroup.Subscription.CloudAccountID.String() != subscriptionID {
				continue
			}
			objects = append(objects, r.Object)
		}
	case "CloudNativeTagRule":
		results, err := hierarchy.ObjectsByName[hierarchy.CloudNativeTagRule](ctx, api, name, hierarchy.WorkloadAllSubHierarchyType, activeFilters...)
		if err != nil {
			res.Diagnostics.AddError("Failed to look up objects", err.Error())
			return
		}

		for _, r := range results {
			objects = append(objects, r.Object)
		}
	case "GitHubOrganization":
		// The inventory query returns a 500 error for the GitHub organization
		// type, so route all GitHub hierarchy lookups through the dedicated
		// queries instead.
		orgs, err := devops.Wrap(polarisClient).GitHubOrganizationsByName(ctx, name,
			activeObjectFilters(hierarchy.Filter{Field: "NAME_EXACT_MATCH", Texts: []string{name}})...)
		if err != nil {
			res.Diagnostics.AddError("Failed to look up GitHub organizations", err.Error())
			return
		}

		for _, org := range orgs {
			objects = append(objects, hierarchy.Object{
				ID:         org.ID,
				Name:       org.Name,
				ObjectType: hierarchy.ObjectType(org.ObjectType),
			})
		}
	case "GitHubRepository":
		// The inventory query returns a 500 error for the GitHub repository
		// type, so route all GitHub hierarchy lookups through the dedicated
		// queries instead.
		repos, err := devops.Wrap(polarisClient).GitHubRepositoriesByName(ctx, name,
			activeObjectFilters(hierarchy.Filter{Field: "NAME_EXACT_MATCH", Texts: []string{name}})...)
		if err != nil {
			res.Diagnostics.AddError("Failed to look up GitHub repositories", err.Error())
			return
		}

		// Repository names are only unique within an organization, so when
		// org_id is set, narrow to that organization.
		orgID := config.OrgID.ValueString()
		for _, repo := range repos {
			if orgID != "" && repo.OrgID.String() != orgID {
				continue
			}
			objects = append(objects, hierarchy.Object{
				ID:         repo.ID,
				Name:       repo.Name,
				ObjectType: hierarchy.ObjectType(repo.ObjectType),
			})
		}
	}

	if len(objects) == 0 {
		res.Diagnostics.AddError("Object not found",
			fmt.Sprintf("no object found with name %q and type %q", name, objectType))
		return
	}
	if len(objects) > 1 {
		res.Diagnostics.AddError("Multiple objects found",
			fmt.Sprintf("multiple objects found with name %q and type %q, try narrowing the search if possible", name, objectType))
		return
	}

	config.ID = types.StringValue(objects[0].ID.String())
	res.Diagnostics.Append(res.State.Set(ctx, config)...)
}

// activeObjectFilters returns the server-side hierarchy filters that exclude
// inactive workload objects: relics, ghosts, and inactive or archived objects.
// Any additional filters passed in are appended to the returned set.
func activeObjectFilters(filters ...hierarchy.Filter) []hierarchy.Filter {
	return append([]hierarchy.Filter{
		{Field: "IS_RELIC", Texts: []string{"false"}},
		{Field: "IS_GHOST", Texts: []string{"false"}},
		{Field: "IS_ACTIVE", Texts: []string{"true"}},
		{Field: "IS_ARCHIVED", Texts: []string{"false"}},
	}, filters...)
}
