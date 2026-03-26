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
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/access"
	gqlaccess "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/access"
)

const dataSourceIdentityProviderDescription = `
The ´polaris_identity_provider´ data source is used to access information about
an identity provider configured in RSC. An identity provider is looked up using
either the ´identity_provider_id´ or the ´name´.

-> **Note:** If multiple identity providers share the same name, look up by name
   will fail. Use the ´identity_provider_id´ attribute to specify the exact
   identity provider.
`

var _ datasource.DataSource = &identityProviderDataSource{}

type identityProviderDataSource struct {
	client *client
}

type identityProviderModel struct {
	ID                   types.String `tfsdk:"id"`
	ActiveUsers          types.Int64  `tfsdk:"active_users"`
	AuthorizedGroups     types.Int64  `tfsdk:"authorized_groups"`
	ClaimAttributes      types.Set    `tfsdk:"claim_attributes"`
	EntityID             types.String `tfsdk:"entity_id"`
	Expiration           types.String `tfsdk:"expiration"`
	IdentityProviderID   types.String `tfsdk:"identity_provider_id"`
	Default              types.Bool   `tfsdk:"default"`
	MetadataJSON         types.String `tfsdk:"metadata_json"`
	Name                 types.String `tfsdk:"name"`
	SignInURL            types.String `tfsdk:"sign_in_url"`
	SigningCertificate   types.String `tfsdk:"signing_certificate"`
	SignOutURL           types.String `tfsdk:"sign_out_url"`
	SPInitiatedSignInURL types.String `tfsdk:"sp_initiated_sign_in_url"`
	SPInitiatedTestURL   types.String `tfsdk:"sp_initiated_test_url"`
}

func newIdentityProviderDataSource() datasource.DataSource {
	return &identityProviderDataSource{}
}

func (d *identityProviderDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, res *datasource.MetadataResponse) {
	tflog.Trace(ctx, "identityProviderDataSource.Metadata")

	res.TypeName = req.ProviderTypeName + "_" + keyIdentityProvider
}

func (d *identityProviderDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, res *datasource.SchemaResponse) {
	tflog.Trace(ctx, "identityProviderDataSource.Schema")

	res.Schema = schema.Schema{
		Description: description(dataSourceIdentityProviderDescription),
		Attributes: map[string]schema.Attribute{
			keyID: schema.StringAttribute{
				Computed:    true,
				Description: "Identity provider ID (UUID).",
			},
			keyActiveUsers: schema.Int64Attribute{
				Computed:    true,
				Description: "Number of active users for this identity provider.",
			},
			keyAuthorizedGroups: schema.Int64Attribute{
				Computed:    true,
				Description: "Number of authorized groups for this identity provider.",
			},
			keyClaimAttributes: schema.SetNestedAttribute{
				Computed:    true,
				Description: "IDP claim attribute mappings.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						keyName: schema.StringAttribute{
							Computed:    true,
							Description: "Claim attribute name.",
						},
						keyAttributeType: schema.StringAttribute{
							Computed:    true,
							Description: "Claim attribute type.",
						},
					},
				},
			},
			keyEntityID: schema.StringAttribute{
				Computed:    true,
				Description: "SAML entity ID.",
			},
			keyExpiration: schema.StringAttribute{
				Computed:    true,
				Description: "Certificate expiration date.",
			},
			keyIdentityProviderID: schema.StringAttribute{
				Optional:    true,
				Description: "Identity provider ID (UUID).",
				Validators: []validator.String{
					isUUID(),
				},
			},
			keyDefault: schema.BoolAttribute{
				Computed:    true,
				Description: "True if this is the default identity provider.",
			},
			keyMetadataJSON: schema.StringAttribute{
				Computed:    true,
				Description: "SAML metadata as JSON.",
			},
			keyName: schema.StringAttribute{
				Optional:    true,
				Description: "Identity provider name.",
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.MatchRoot(keyIdentityProviderID)),
					isNotWhiteSpace(),
				},
			},
			keySignInURL: schema.StringAttribute{
				Computed:    true,
				Description: "SAML sign-in URL.",
			},
			keySigningCertificate: schema.StringAttribute{
				Computed:    true,
				Description: "SAML signing certificate.",
				Sensitive:   true,
			},
			keySignOutURL: schema.StringAttribute{
				Computed:    true,
				Description: "SAML sign-out URL.",
			},
			keySPInitiatedSignInURL: schema.StringAttribute{
				Computed:    true,
				Description: "Service provider initiated sign-in URL.",
			},
			keySPInitiatedTestURL: schema.StringAttribute{
				Computed:    true,
				Description: "Service provider initiated test URL.",
			},
		},
	}
}

func (d *identityProviderDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, res *datasource.ConfigureResponse) {
	tflog.Trace(ctx, "identityProviderDataSource.Configure")

	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*client)
}

func (d *identityProviderDataSource) Read(ctx context.Context, req datasource.ReadRequest, res *datasource.ReadResponse) {
	tflog.Trace(ctx, "identityProviderDataSource.Read")

	var config identityProviderModel
	res.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if res.Diagnostics.HasError() {
		return
	}

	polarisClient, err := d.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	var idp gqlaccess.IdentityProvider
	if !config.IdentityProviderID.IsNull() {
		idp, err = access.Wrap(polarisClient).IdentityProviderByID(ctx, config.IdentityProviderID.ValueString())
		if err != nil {
			res.Diagnostics.AddError("Failed to read identity provider", err.Error())
			return
		}
	} else {
		idp, err = access.Wrap(polarisClient).IdentityProviderByName(ctx, config.Name.ValueString())
		if err != nil {
			res.Diagnostics.AddError("Failed to read identity provider", err.Error())
			return
		}
	}

	claimsSet, diags := fromIDPClaimAttributes(idp.ClaimAttributes)
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}

	expiration := types.StringNull()
	if !idp.Expiration.IsZero() {
		expiration = types.StringValue(idp.Expiration.Format(time.RFC3339))
	}

	state := identityProviderModel{
		ID:                   types.StringValue(idp.ID),
		ActiveUsers:          types.Int64Value(int64(idp.ActiveUsers)),
		AuthorizedGroups:     types.Int64Value(int64(idp.AuthorizedGroups)),
		ClaimAttributes:      claimsSet,
		EntityID:             types.StringValue(idp.EntityID),
		Expiration:           expiration,
		IdentityProviderID:   types.StringValue(idp.ID),
		Default:              types.BoolValue(idp.Default),
		MetadataJSON:         types.StringValue(idp.MetadataJSON),
		Name:                 types.StringValue(idp.Name),
		SignInURL:            types.StringValue(idp.SignInURL),
		SigningCertificate:   types.StringValue(idp.SigningCertificate),
		SignOutURL:           types.StringValue(idp.SignOutURL),
		SPInitiatedSignInURL: types.StringValue(idp.SPInitiatedSignInURL),
		SPInitiatedTestURL:   types.StringValue(idp.SPInitiatedTestURL),
	}

	res.Diagnostics.Append(res.State.Set(ctx, &state)...)
}

// idpClaimAttrTypes returns the attribute types for the claim attribute nested
// set.
func idpClaimAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		keyName:          types.StringType,
		keyAttributeType: types.StringType,
	}
}

// fromIDPClaimAttributes converts a slice of claim attributes to a Terraform
// Framework set.
func fromIDPClaimAttributes(claims []struct {
	Name string `json:"name"`
	Type string `json:"attributeType"`
}) (types.Set, diag.Diagnostics) {
	claimValues := make([]attr.Value, 0, len(claims))
	for _, claim := range claims {
		claimValue, diags := types.ObjectValue(idpClaimAttrTypes(), map[string]attr.Value{
			keyName:          types.StringValue(claim.Name),
			keyAttributeType: types.StringValue(claim.Type),
		})
		if diags.HasError() {
			return types.SetNull(types.ObjectType{AttrTypes: idpClaimAttrTypes()}), diags
		}
		claimValues = append(claimValues, claimValue)
	}

	return types.SetValue(types.ObjectType{AttrTypes: idpClaimAttrTypes()}, claimValues)
}
