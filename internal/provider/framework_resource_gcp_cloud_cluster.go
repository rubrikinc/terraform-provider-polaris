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
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/cloudcluster"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/cluster"
	gqlcloudcluster "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/cloudcluster"
	gqlcluster "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/cluster"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core/secret"
)

const gcpServiceAccountScope = "https://www.googleapis.com/auth/cloud-platform"

const defaultGCPCloudClusterCreateTimeout = 60 * time.Minute

// minMultiAzNodes is the minimum node count required for a Multi-AZ cluster.
const minMultiAzNodes = 3

// gcpCloudClusterInstanceTypes are the GCP instance types accepted by the
// instance_type attribute. The backend further restricts the set per CDM
// version; this is the superset the provider allows at plan time. It drives both
// the OneOf validator and the attribute description so they cannot drift.
var gcpCloudClusterInstanceTypes = []gqlcloudcluster.GcpCCInstanceType{
	gqlcloudcluster.GcpInstanceTypeN2Standard8,
	gqlcloudcluster.GcpInstanceTypeN2Standard16,
	gqlcloudcluster.GcpInstanceTypeN2Highmem16,
	gqlcloudcluster.GcpInstanceTypeN2DStandard8,
	gqlcloudcluster.GcpInstanceTypeN2DStandard16,
	gqlcloudcluster.GcpInstanceTypeN2DHighmem16,
}

const resourceGCPCloudClusterDescription = `
The ´polaris_gcp_cloud_cluster´ resource creates a GCP cloud cluster using RSC.

This resource creates a Rubrik Cloud Data Management (CDM) cluster with elastic storage
in GCP using the specified configuration. The cluster will be deployed with the specified
number of nodes, instance types, and network configuration.

~> **Note:** This resource creates actual GCP infrastructure. Destroying the
   resource will attempt to clean up the created resources, but manual cleanup
   may be required.

~> **Note:** The GCP project must be onboarded to RSC with the ´SERVERS_AND_APPS´
   feature enabled before creating a cloud cluster.

~> **Note:** This resource requires **Terraform v1.11.0 or later** due to the use of write-only attributes for
   ´admin_email´ and ´admin_password´.
`

var (
	_ resource.Resource                   = &gcpCloudClusterResource{}
	_ resource.ResourceWithConfigure      = &gcpCloudClusterResource{}
	_ resource.ResourceWithValidateConfig = &gcpCloudClusterResource{}
)

type gcpCloudClusterResource struct {
	client *client
}

type gcpCloudClusterModel struct {
	ID             types.String            `tfsdk:"id"`
	CloudAccountID types.String            `tfsdk:"cloud_account_id"`
	Region         types.String            `tfsdk:"region"`
	Zone           types.String            `tfsdk:"zone"`
	AZResilient    types.Bool              `tfsdk:"az_resilient"`
	ClusterConfig  []gcpClusterConfigModel `tfsdk:"cluster_config"`
	VMConfig       []gcpVMConfigModel      `tfsdk:"vm_config"`
	Timeouts       timeouts.Value          `tfsdk:"timeouts"`
}

type gcpClusterConfigModel struct {
	ClusterName                 types.String `tfsdk:"cluster_name"`
	AdminEmail                  types.String `tfsdk:"admin_email"`
	AdminPassword               types.String `tfsdk:"admin_password"`
	NumNodes                    types.Int64  `tfsdk:"num_nodes"`
	DNSNameServers              types.Set    `tfsdk:"dns_name_servers"`
	DNSSearchDomains            types.Set    `tfsdk:"dns_search_domains"`
	NTPServers                  types.Set    `tfsdk:"ntp_servers"`
	BucketName                  types.String `tfsdk:"bucket_name"`
	KeepClusterOnFailure        types.Bool   `tfsdk:"keep_cluster_on_failure"`
	ForceClusterDeleteOnDestroy types.Bool   `tfsdk:"force_cluster_delete_on_destroy"`
	Timezone                    types.String `tfsdk:"timezone"`
	Location                    types.String `tfsdk:"location"`
}

type gcpVMConfigModel struct {
	CDMVersion      types.String `tfsdk:"cdm_version"`
	CDMProduct      types.String `tfsdk:"cdm_product"`
	InstanceType    types.String `tfsdk:"instance_type"`
	Network         types.String `tfsdk:"network"`
	Subnet          types.String `tfsdk:"subnet"`
	HostProject     types.String `tfsdk:"host_project"`
	ServiceAccounts types.Set    `tfsdk:"service_accounts"`
	// SubnetAzConfig is a types.List (not []gcpSubnetAzConfigModel) so it can hold
	// an unknown value: it is commonly built from a dynamic block whose for_each
	// derives from a data source, which is unknown until the data source is read.
	// A Go slice target cannot decode an unknown and would fail ValidateConfig.
	SubnetAzConfig   types.List `tfsdk:"subnet_az_config"`
	DeleteProtection types.Bool `tfsdk:"delete_protection"`
}

type gcpSubnetAzConfigModel struct {
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	Subnet           types.String `tfsdk:"subnet"`
}

func newGcpCloudClusterResource() resource.Resource {
	return &gcpCloudClusterResource{}
}

func (r *gcpCloudClusterResource) Metadata(ctx context.Context, req resource.MetadataRequest, res *resource.MetadataResponse) {
	tflog.Trace(ctx, "gcpCloudClusterResource.Metadata")

	res.TypeName = req.ProviderTypeName + "_gcp_cloud_cluster"
}

func (r *gcpCloudClusterResource) Schema(ctx context.Context, _ resource.SchemaRequest, res *resource.SchemaResponse) {
	tflog.Trace(ctx, "gcpCloudClusterResource.Schema")

	requiresReplaceStr := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	useStateStr := []planmodifier.String{stringplanmodifier.UseStateForUnknown()}

	instanceTypeValues := make([]string, len(gcpCloudClusterInstanceTypes))
	for i, t := range gcpCloudClusterInstanceTypes {
		instanceTypeValues[i] = string(t)
	}

	res.Schema = schema.Schema{
		Description: description(resourceGCPCloudClusterDescription),
		Attributes: map[string]schema.Attribute{
			keyID: schema.StringAttribute{
				Computed:      true,
				Description:   "Cloud cluster ID (UUID).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			keyCloudAccountID: schema.StringAttribute{
				Required:      true,
				Description:   "RSC cloud account ID (UUID). Changing this forces a new resource to be created.",
				PlanModifiers: requiresReplaceStr,
				Validators:    []validator.String{isUUID()},
			},
			keyRegion: schema.StringAttribute{
				Required:      true,
				Description:   "GCP region to deploy the cluster in. Changing this forces a new resource to be created.",
				PlanModifiers: requiresReplaceStr,
				Validators:    []validator.String{isNotWhiteSpace()},
			},
			keyZone: schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "GCP zone to deploy the cluster in. Required when `az_resilient` is false. When `az_resilient` is true it may be omitted and defaults to the first `subnet_az_config` availability zone. Changing this forces a new resource to be created.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
				Validators:    []validator.String{isNotWhiteSpace()},
			},
			keyAzResilient: schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Default:       booldefault.StaticBool(false),
				Description:   "Whether to deploy the cluster across multiple availability zones for AZ resiliency. When enabled, `subnet_az_config` blocks must be specified in `vm_config` and `subnet` must be omitted. Requires at least three nodes and a region with at least three zones. Changing this forces a new resource to be created.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
		},
		Blocks: map[string]schema.Block{
			keyClusterConfig: schema.ListNestedBlock{
				Description: "Configuration for the cloud cluster.",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.SizeAtMost(1),
					listvalidator.IsRequired(),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						keyClusterName: schema.StringAttribute{
							Required:    true,
							Description: "Unique name to assign to the cloud cluster.",
							Validators:  []validator.String{isNotWhiteSpace()},
						},
						keyAdminEmail: schema.StringAttribute{
							Required:    true,
							WriteOnly:   true,
							Description: "Email address for the cluster admin user. Changing this value will have no effect on the cluster.",
							Validators:  []validator.String{isNotWhiteSpace()},
						},
						keyAdminPassword: schema.StringAttribute{
							Required:    true,
							Sensitive:   true,
							WriteOnly:   true,
							Description: "Password for the cluster admin user. Changing this value will have no effect on the cluster.",
							Validators:  []validator.String{isNotWhiteSpace()},
						},
						keyNumNodes: schema.Int64Attribute{
							Required:      true,
							Description:   "Number of nodes in the cluster. Must be at least 3 when `az_resilient` is true. Changing this forces a new resource to be created.",
							PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
							Validators:    []validator.Int64{int64validator.AtLeast(1)},
						},
						keyDNSNameServers: schema.SetAttribute{
							ElementType: types.StringType,
							Required:    true,
							Description: "DNS name servers for the cluster.",
							Validators:  []validator.Set{setvalidator.SizeAtLeast(1)},
						},
						keyDNSSearchDomains: schema.SetAttribute{
							ElementType:   types.StringType,
							Optional:      true,
							Computed:      true,
							Description:   "DNS search domains for the cluster.",
							PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
						},
						keyNTPServers: schema.SetAttribute{
							ElementType: types.StringType,
							Required:    true,
							Description: "NTP servers for the cluster.",
							Validators:  []validator.Set{setvalidator.SizeAtLeast(1)},
						},
						keyBucketName: schema.StringAttribute{
							Required:      true,
							Description:   "Name of the GCS bucket to use for the cluster. Changing this forces a new resource to be created.",
							PlanModifiers: requiresReplaceStr,
							Validators:    []validator.String{isNotWhiteSpace()},
						},
						keyKeepClusterOnFailure: schema.BoolAttribute{
							Required:      true,
							Description:   "Whether to keep the cluster on failure (can be useful for troubleshooting). Changing this forces a new resource to be created.",
							PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
						},
						keyForceClusterDeleteOnDestroy: schema.BoolAttribute{
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
							Description: "Whether to force delete the cluster on destroy.",
						},
						keyTimezone: schema.StringAttribute{
							Optional:      true,
							Computed:      true,
							Description:   "Timezone for the cluster using IANA standard format e.g. America/Los_Angeles, Europe/Paris, etc.",
							PlanModifiers: useStateStr,
							Validators:    []validator.String{isNotWhiteSpace()},
						},
						keyLocation: schema.StringAttribute{
							Optional:      true,
							Computed:      true,
							Description:   "Location for the cluster. This is free text, RSC will map it to the closest possible location e.g. Palo Alto, CA.",
							PlanModifiers: useStateStr,
							Validators:    []validator.String{isNotWhiteSpace()},
						},
					},
				},
			},
			keyVMConfig: schema.ListNestedBlock{
				Description: "VM configuration for the cluster nodes. Changing this forces a new resource to be created.",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.SizeAtMost(1),
					listvalidator.IsRequired(),
				},
				PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace()},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						keyCDMVersion: schema.StringAttribute{
							Required:    true,
							Description: "CDM version to use. Changing this forces a new resource to be created.",
							Validators:  []validator.String{isNotWhiteSpace()},
						},
						keyCDMProduct: schema.StringAttribute{
							Computed:      true,
							Description:   "CDM Product Code. This is a read-only field and computed based on the CDM version.",
							PlanModifiers: useStateStr,
						},
						keyInstanceType: schema.StringAttribute{
							Required: true,
							Description: fmt.Sprintf("GCP instance type for the cluster nodes. Changing this forces a new resource to be created. %s. "+
								"The set of instance types actually available depends on the selected CDM version.", possibleValues(gcpCloudClusterInstanceTypes)),
							Validators: []validator.String{stringvalidator.OneOf(instanceTypeValues...)},
						},
						keyNetwork: schema.StringAttribute{
							Required:    true,
							Description: "GCP network name for the cluster nodes. Changing this forces a new resource to be created.",
							Validators:  []validator.String{isNotWhiteSpace()},
						},
						keySubnet: schema.StringAttribute{
							Optional:    true,
							Description: "GCP subnet name for the cluster nodes. Required when `az_resilient` is false; omit it and use `subnet_az_config` when `az_resilient` is true. Changing this forces a new resource to be created.",
							Validators:  []validator.String{isNotWhiteSpace()},
						},
						keyHostProject: schema.StringAttribute{
							Optional:    true,
							Description: "GCP host project for shared VPC. Changing this forces a new resource to be created.",
						},
						keyServiceAccounts: schema.SetAttribute{
							ElementType: types.StringType,
							Required:    true,
							Description: "GCP service account emails for the cluster nodes. Changing this forces a new resource to be created.",
							Validators:  []validator.Set{setvalidator.SizeAtLeast(1)},
						},
						keyDeleteProtection: schema.BoolAttribute{
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(true),
							Description: "Whether to enable delete protection on the GCP instances. Changing this forces a new resource to be created.",
						},
					},
					Blocks: map[string]schema.Block{
						keySubnetAzConfigs: schema.ListNestedBlock{
							Description: "Subnet and availability zone pairs for Multi-AZ deployments. Required when `az_resilient` is true. Each block specifies a subnet and its availability zone; the network and host project are taken from the `network` and `host_project` fields. Changing this forces a new resource to be created.",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									keyAvailabilityZone: schema.StringAttribute{
										Required:    true,
										Description: "Availability zone name, e.g. `us-west1-a`.",
										Validators:  []validator.String{isNotWhiteSpace()},
									},
									keySubnet: schema.StringAttribute{
										Required:    true,
										Description: "GCP subnet name for this availability zone.",
										Validators:  []validator.String{isNotWhiteSpace()},
									},
								},
							},
						},
					},
				},
			},
			// Only Create is wired up (the create/read-back is the long-running
			// step); don't advertise timeouts the resource doesn't apply.
			keyTimeouts: timeouts.Block(ctx, timeouts.Opts{
				Create: true,
			}),
		},
	}
}

func (r *gcpCloudClusterResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, res *resource.ValidateConfigResponse) {
	tflog.Trace(ctx, "gcpCloudClusterResource.ValidateConfig")

	var config gcpCloudClusterModel
	res.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if res.Diagnostics.HasError() {
		return
	}

	res.Diagnostics.Append(validateGcpCloudClusterConfig(config)...)
}

// validateGcpCloudClusterConfig holds the plan-time, client-free validation
// rules: the az_resilient / subnet / subnet_az_config either-or, and the
// Multi-AZ minimum node count.
//
// Checks are skipped for any value that is unknown at plan time (an unresolved
// reference known only after apply), because reading it would yield a zero
// value and reject an otherwise-valid config.
func validateGcpCloudClusterConfig(config gcpCloudClusterModel) diag.Diagnostics {
	var diags diag.Diagnostics

	if len(config.VMConfig) == 0 || len(config.ClusterConfig) == 0 {
		return diags
	}
	vm := config.VMConfig[0]

	// az_resilient decides which subnet form is required, so nothing here is
	// conclusive until it is known.
	if config.AZResilient.IsUnknown() {
		return diags
	}
	azResilient := config.AZResilient.ValueBool()

	// subnet and subnet_az_config are only conclusive once known. subnet_az_config
	// is frequently built from a data-source-driven dynamic block, so it can be
	// unknown at validate time.
	subnetAzKnown := !vm.SubnetAzConfig.IsUnknown()
	hasSubnetAzConfigs := subnetAzKnown && !vm.SubnetAzConfig.IsNull() && len(vm.SubnetAzConfig.Elements()) > 0
	subnetKnown := !vm.Subnet.IsUnknown()
	hasSubnet := subnetKnown && vm.Subnet.ValueString() != ""

	if azResilient {
		if subnetAzKnown && !hasSubnetAzConfigs {
			diags.AddAttributeError(path.Root(keyVMConfig), "subnet_az_config required",
				"`subnet_az_config` is required in `vm_config` when `az_resilient` is true.")
		}
		if hasSubnet {
			diags.AddAttributeError(path.Root(keyVMConfig), "subnet not allowed",
				"`subnet` cannot be specified in `vm_config` when `az_resilient` is true, use `subnet_az_config` instead.")
		}
		if n := config.ClusterConfig[0].NumNodes; !n.IsUnknown() && !n.IsNull() && n.ValueInt64() < minMultiAzNodes {
			diags.AddAttributeError(path.Root(keyClusterConfig), "num_nodes too low",
				fmt.Sprintf("`num_nodes` must be at least %d when `az_resilient` is true.", minMultiAzNodes))
		}
	} else {
		if hasSubnetAzConfigs {
			diags.AddAttributeError(path.Root(keyVMConfig), "subnet_az_config not allowed",
				"`subnet_az_config` cannot be specified in `vm_config` when `az_resilient` is false.")
		}
		if subnetKnown && !hasSubnet {
			diags.AddAttributeError(path.Root(keyVMConfig), "subnet required",
				"`subnet` is required in `vm_config` when `az_resilient` is false.")
		}
		// zone is only derived from subnet_az_config in Multi-AZ mode, so it must
		// be set for single-AZ clusters.
		if config.Zone.IsNull() {
			diags.AddAttributeError(path.Root(keyZone), "zone required",
				"`zone` is required when `az_resilient` is false.")
		}
	}

	return diags
}

func (r *gcpCloudClusterResource) Configure(ctx context.Context, req resource.ConfigureRequest, res *resource.ConfigureResponse) {
	tflog.Trace(ctx, "gcpCloudClusterResource.Configure")

	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*client)
}

func (r *gcpCloudClusterResource) Create(ctx context.Context, req resource.CreateRequest, res *resource.CreateResponse) {
	tflog.Trace(ctx, "gcpCloudClusterResource.Create")

	var plan gcpCloudClusterModel
	res.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if res.Diagnostics.HasError() {
		return
	}

	// admin_email and admin_password are write-only, so they are null in the
	// plan; read them from the config instead.
	var config gcpCloudClusterModel
	res.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if res.Diagnostics.HasError() {
		return
	}

	timeout, diags := plan.Timeouts.Create(ctx, defaultGCPCloudClusterCreateTimeout)
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	polarisClient, err := r.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	input, diags := r.buildCreateInput(ctx, plan, config)
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}

	gcpCluster, err := cloudcluster.Wrap(polarisClient).CreateGcpCloudCluster(ctx, input, true)
	if err != nil {
		res.Diagnostics.AddError("Failed to create GCP cloud cluster", err.Error())
		return
	}

	plan.ID = types.StringValue(gcpCluster.ID.String())
	plan.VMConfig[0].CDMProduct = types.StringValue(gcpCluster.CdmProduct)
	// zone is Computed and may have been derived from subnet_az_config, so record
	// the resolved value in state.
	plan.Zone = types.StringValue(input.Zone)

	// Read back to populate computed fields. A failed readback must not fail the
	// create; the cluster exists and a plan diff on the next run is acceptable.
	if diags := r.refresh(ctx, polarisClient, &plan); diags.HasError() {
		for _, d := range diags {
			tflog.Warn(ctx, "failed to read back gcp cloud cluster after create", map[string]any{
				"summary": d.Summary(), "detail": d.Detail(),
			})
		}
	}

	// If the readback failed, Optional+Computed fields may still be unknown,
	// which cannot be stored in state. Coerce any leftover unknowns to null so
	// the successfully-created resource is still saved.
	cc := &plan.ClusterConfig[0]
	if cc.Timezone.IsUnknown() {
		cc.Timezone = types.StringNull()
	}
	if cc.Location.IsUnknown() {
		cc.Location = types.StringNull()
	}
	if cc.DNSSearchDomains.IsUnknown() {
		cc.DNSSearchDomains = types.SetNull(types.StringType)
	}

	// Write-only attributes must be null in state.
	plan.ClusterConfig[0].AdminEmail = types.StringNull()
	plan.ClusterConfig[0].AdminPassword = types.StringNull()

	res.Diagnostics.Append(res.State.Set(ctx, &plan)...)
}

func (r *gcpCloudClusterResource) Read(ctx context.Context, req resource.ReadRequest, res *resource.ReadResponse) {
	tflog.Trace(ctx, "gcpCloudClusterResource.Read")

	var state gcpCloudClusterModel
	res.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if res.Diagnostics.HasError() {
		return
	}

	polarisClient, err := r.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	removed, diags := r.refreshExisting(ctx, polarisClient, &state)
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}
	if removed {
		res.State.RemoveResource(ctx)
		return
	}

	res.Diagnostics.Append(res.State.Set(ctx, &state)...)
}

func (r *gcpCloudClusterResource) Update(ctx context.Context, req resource.UpdateRequest, res *resource.UpdateResponse) {
	tflog.Trace(ctx, "gcpCloudClusterResource.Update")

	var plan, state gcpCloudClusterModel
	res.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	res.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if res.Diagnostics.HasError() {
		return
	}

	clusterID, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		res.Diagnostics.AddError("Invalid cloud cluster ID", err.Error())
		return
	}

	polarisClient, err := r.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}
	gqlClusterAPI := gqlcluster.Wrap(polarisClient.GQL)

	planCC := plan.ClusterConfig[0]
	stateCC := state.ClusterConfig[0]

	// Only call each update API when its fields actually changed. Calling them
	// unconditionally sends no-op updates that the backend can reject.

	// DNS name servers and search domains are updated together.
	if !planCC.DNSNameServers.Equal(stateCC.DNSNameServers) || !planCC.DNSSearchDomains.Equal(stateCC.DNSSearchDomains) {
		dnsServers := decodeStringSetOrNil(ctx, planCC.DNSNameServers, &res.Diagnostics)
		searchDomains := decodeStringSetOrNil(ctx, planCC.DNSSearchDomains, &res.Diagnostics)
		if res.Diagnostics.HasError() {
			return
		}
		if err := gqlClusterAPI.UpdateDNSServersAndSearchDomains(ctx, gqlcluster.UpdateDNSServersAndSearchDomainsInput{
			ClusterID:     clusterID,
			DNSServers:    dnsServers,
			SearchDomains: searchDomains,
		}); err != nil {
			res.Diagnostics.AddError("Failed to update DNS servers and search domains", err.Error())
			return
		}
	}

	if !planCC.NTPServers.Equal(stateCC.NTPServers) {
		ntpServers := decodeStringSetOrNil(ctx, planCC.NTPServers, &res.Diagnostics)
		if res.Diagnostics.HasError() {
			return
		}
		ntpInput := gqlcluster.UpdateClusterNTPServersInput{ClusterID: clusterID}
		for _, ntp := range ntpServers {
			ntpInput.Servers = append(ntpInput.Servers, struct {
				Server       string                      `json:"server"`
				SymmetricKey *gqlcluster.NTPSymmetricKey `json:"symmetricKey,omitempty"`
			}{Server: ntp})
		}
		if err := gqlClusterAPI.UpdateNTPServers(ctx, ntpInput); err != nil {
			res.Diagnostics.AddError("Failed to update NTP servers", err.Error())
			return
		}
	}

	// Cluster name, timezone and location are updated together via one API.
	if !planCC.ClusterName.Equal(stateCC.ClusterName) || !planCC.Timezone.Equal(stateCC.Timezone) || !planCC.Location.Equal(stateCC.Location) {
		var parsedTimezone gqlcluster.Timezone
		if tz := planCC.Timezone.ValueString(); tz != "" {
			parsedTimezone, err = gqlcluster.ParseTimeZone(tz)
			if err != nil {
				res.Diagnostics.AddError("Invalid timezone", err.Error())
				return
			}
		}
		if _, err := gqlClusterAPI.UpdateSettings(ctx, gqlcluster.UpdatedSettings{
			ClusterID: clusterID,
			Name:      planCC.ClusterName.ValueString(),
			Timezone:  parsedTimezone,
			Address:   planCC.Location.ValueString(),
		}); err != nil {
			res.Diagnostics.AddError("Failed to update cluster settings", err.Error())
			return
		}
	}

	if diags := r.refresh(ctx, polarisClient, &plan); diags.HasError() {
		res.Diagnostics.Append(diags...)
		return
	}

	// Write-only attributes must be null in state.
	plan.ClusterConfig[0].AdminEmail = types.StringNull()
	plan.ClusterConfig[0].AdminPassword = types.StringNull()

	res.Diagnostics.Append(res.State.Set(ctx, &plan)...)
}

func (r *gcpCloudClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, res *resource.DeleteResponse) {
	tflog.Trace(ctx, "gcpCloudClusterResource.Delete")

	var state gcpCloudClusterModel
	res.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if res.Diagnostics.HasError() {
		return
	}

	clusterID, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		res.Diagnostics.AddError("Invalid cloud cluster ID", err.Error())
		return
	}

	polarisClient, err := r.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	forceRemoval := false
	if len(state.ClusterConfig) > 0 {
		forceRemoval = state.ClusterConfig[0].ForceClusterDeleteOnDestroy.ValueBool()
	}

	if _, err := cluster.Wrap(polarisClient).RemoveCluster(ctx, clusterID, forceRemoval, 0); err != nil {
		res.Diagnostics.AddError("Failed to remove GCP cloud cluster", err.Error())
		return
	}
}

// buildCreateInput assembles the SDK CreateGcpClusterInput from the plan (for
// non-write-only fields) and the config (for the write-only admin fields).
func (r *gcpCloudClusterResource) buildCreateInput(ctx context.Context, plan, config gcpCloudClusterModel) (gqlcloudcluster.CreateGcpClusterInput, diag.Diagnostics) {
	var diags diag.Diagnostics

	cloudAccountID, err := uuid.Parse(plan.CloudAccountID.ValueString())
	if err != nil {
		diags.AddError("Invalid cloud account ID", err.Error())
		return gqlcloudcluster.CreateGcpClusterInput{}, diags
	}

	cc := plan.ClusterConfig[0]
	vm := plan.VMConfig[0]
	region := plan.Region.ValueString()
	numNodes := int(cc.NumNodes.ValueInt64())
	azResilient := plan.AZResilient.ValueBool()

	var subnetAzModels []gcpSubnetAzConfigModel
	if !vm.SubnetAzConfig.IsNull() && !vm.SubnetAzConfig.IsUnknown() {
		diags.Append(vm.SubnetAzConfig.ElementsAs(ctx, &subnetAzModels, false)...)
		if diags.HasError() {
			return gqlcloudcluster.CreateGcpClusterInput{}, diags
		}
	}
	var subnetAzConfigs []gqlcloudcluster.SubnetAzConfig
	for _, az := range subnetAzModels {
		subnetAzConfigs = append(subnetAzConfigs, gqlcloudcluster.SubnetAzConfig{
			AvailabilityZone: az.AvailabilityZone.ValueString(),
			Subnet:           az.Subnet.ValueString(),
		})
	}

	// The backend requires networkConfig to be populated in both modes. For a
	// Multi-AZ cluster the base subnet comes from the first subnet_az_config
	// entry (network and host project are shared); otherwise it is the single
	// subnet field. The SDK fans networkConfig[0] out to one entry per node.
	subnet := vm.Subnet.ValueString()
	if azResilient && len(subnetAzConfigs) > 0 {
		subnet = subnetAzConfigs[0].Subnet
	}

	// zone defaults to the first subnet_az_config availability zone for a
	// Multi-AZ cluster when the caller omits it, matching the RSC UI.
	zone := plan.Zone.ValueString()
	if azResilient && zone == "" && len(subnetAzConfigs) > 0 {
		zone = subnetAzConfigs[0].AvailabilityZone
	}
	networkConfig := make([]gqlcloudcluster.GcpSubnetInput, numNodes)
	for i := 0; i < numNodes; i++ {
		networkConfig[i] = gqlcloudcluster.GcpSubnetInput{
			HostProject: vm.HostProject.ValueString(),
			Name:        subnet,
			Network:     vm.Network.ValueString(),
			Region:      region,
		}
	}

	serviceAccountEmails := decodeStringSetOrNil(ctx, vm.ServiceAccounts, &diags)
	dnsNameServers := decodeStringSetOrNil(ctx, cc.DNSNameServers, &diags)
	dnsSearchDomains := decodeStringSetOrNil(ctx, cc.DNSSearchDomains, &diags)
	ntpServers := decodeStringSetOrNil(ctx, cc.NTPServers, &diags)
	if diags.HasError() {
		return gqlcloudcluster.CreateGcpClusterInput{}, diags
	}
	if dnsSearchDomains == nil {
		dnsSearchDomains = []string{}
	}

	serviceAccounts := make([]gqlcloudcluster.GcpServiceAccountInput, 0, len(serviceAccountEmails))
	for _, sa := range serviceAccountEmails {
		serviceAccounts = append(serviceAccounts, gqlcloudcluster.GcpServiceAccountInput{
			Email:  sa,
			Scopes: []string{gcpServiceAccountScope},
		})
	}

	// admin_email and admin_password are write-only, so read them from the config.
	configCC := config.ClusterConfig[0]

	var isAzResilient *bool
	if azResilient {
		isAzResilient = &azResilient
	}

	input := gqlcloudcluster.CreateGcpClusterInput{
		CloudAccountID:       cloudAccountID,
		IsEsType:             true,
		IsAzResilient:        isAzResilient,
		KeepClusterOnFailure: cc.KeepClusterOnFailure.ValueBool(),
		Region:               region,
		Zone:                 zone,
		Validations:          []gqlcloudcluster.ClusterCreateValidations{gqlcloudcluster.AllChecks},
		ClusterConfig: gqlcloudcluster.GcpClusterConfig{
			ClusterName:      cc.ClusterName.ValueString(),
			UserEmail:        configCC.AdminEmail.ValueString(),
			AdminPassword:    secret.String(configCC.AdminPassword.ValueString()),
			DNSNameServers:   dnsNameServers,
			DNSSearchDomains: dnsSearchDomains,
			NTPServers:       ntpServers,
			NumNodes:         numNodes,
			GcpEsConfig: gqlcloudcluster.GcpEsConfigInput{
				BucketName:         cc.BucketName.ValueString(),
				Region:             region,
				ShouldCreateBucket: false,
			},
		},
		VMConfig: gqlcloudcluster.GcpVmConfig{
			CDMVersion:       vm.CDMVersion.ValueString(),
			InstanceType:     gqlcloudcluster.GcpCCInstanceType(vm.InstanceType.ValueString()),
			NetworkConfig:    networkConfig,
			ServiceAccounts:  serviceAccounts,
			SubnetAzConfigs:  subnetAzConfigs,
			DeleteProtection: vm.DeleteProtection.ValueBool(),
		},
	}

	return input, diags
}

// refresh re-reads the cluster and updates the model's computed fields. It
// treats a missing cluster as an error; use refreshExisting from Read.
func (r *gcpCloudClusterResource) refresh(ctx context.Context, polarisClient *polaris.Client, model *gcpCloudClusterModel) diag.Diagnostics {
	removed, diags := r.refreshExisting(ctx, polarisClient, model)
	if !diags.HasError() && removed {
		diags.AddError("Cloud cluster not found", "the cloud cluster no longer exists in RSC")
	}
	return diags
}

// refreshExisting re-reads the cluster and updates the model's computed fields
// from RSC. It returns removed=true when the cluster no longer exists.
func (r *gcpCloudClusterResource) refreshExisting(ctx context.Context, polarisClient *polaris.Client, model *gcpCloudClusterModel) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	id, err := uuid.Parse(model.ID.ValueString())
	if err != nil {
		diags.AddError("Invalid cloud cluster ID", err.Error())
		return false, diags
	}

	clusterFilter := gqlcluster.SearchFilter{ID: []string{id.String()}}
	cloudClusters, err := cluster.Wrap(polarisClient).ListClusters(ctx, clusterFilter, gqlcluster.SortByClusterName, core.SortOrderDesc)
	if err != nil {
		diags.AddError("Failed to read GCP cloud cluster", err.Error())
		return false, diags
	}
	if len(cloudClusters) == 0 {
		return true, diags
	}
	cc := cloudClusters[0]

	gqlClusterAPI := gqlcluster.Wrap(polarisClient.GQL)

	dnsServers, err := gqlClusterAPI.DNSServers(ctx, id)
	if err != nil {
		diags.AddError("Failed to read DNS servers", err.Error())
		return false, diags
	}
	ntpServers, err := gqlClusterAPI.NTPServers(ctx, id)
	if err != nil {
		diags.AddError("Failed to read NTP servers", err.Error())
		return false, diags
	}
	settings, err := gqlClusterAPI.ClusterSettings(ctx, id)
	if err != nil {
		diags.AddError("Failed to read cluster settings", err.Error())
		return false, diags
	}

	dnsSet, d := types.SetValueFrom(ctx, types.StringType, dnsServers.Servers)
	diags.Append(d...)
	searchSet, d := types.SetValueFrom(ctx, types.StringType, dnsServers.Domains)
	diags.Append(d...)
	ntpValues := make([]string, 0, len(ntpServers))
	for _, s := range ntpServers {
		ntpValues = append(ntpValues, s.Server)
	}
	ntpSet, d := types.SetValueFrom(ctx, types.StringType, ntpValues)
	diags.Append(d...)
	if diags.HasError() {
		return false, diags
	}

	model.ClusterConfig[0].ClusterName = types.StringValue(settings.Name)
	model.ClusterConfig[0].Timezone = types.StringValue(settings.Timezone)
	model.ClusterConfig[0].Location = types.StringValue(settings.RawAddress)
	model.ClusterConfig[0].DNSNameServers = dnsSet
	model.ClusterConfig[0].DNSSearchDomains = searchSet
	model.ClusterConfig[0].NTPServers = ntpSet
	model.VMConfig[0].CDMVersion = types.StringValue(cc.Version)

	return false, diags
}
