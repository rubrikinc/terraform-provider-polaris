// Copyright 2024 Rubrik, Inc.
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

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	gqlsla "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/sla"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/sla"
)

const dataSourceSLADomainDescription = `
The ´polaris_sla_domain´ data source is used to access information about RSC SLA
domains. A SLA domain is looked up using either the ID or the name.
`

const (
	keyArchivalSpec  = "archival_spec"
	keyThreshold     = "threshold"
	keyThresholdUnit = "threshold_unit"
	keyFrequencies   = "frequencies"
)

func dataSourceSLADomain() *schema.Resource {
	return &schema.Resource{
		ReadContext: slaDomainRead,

		Description: description(dataSourceSLADomainDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{keyName},
				Description:  "SLA domain ID (UUID).",
				ValidateFunc: validation.IsUUID,
			},
			keyDescription: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "SLA domain description.",
			},
			keyName: {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{keyID},
				Description:  "SLA domain name.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyObjectTypes: {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "Object types which can be protected by the SLA domain.",
			},
			keyArchivalSpec: {
				Type:        schema.TypeSet,
				Elem:        archivalSpecResource(),
				Computed:    true,
				Description: "",
			},
			keyDailySchedule: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyFrequency: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Frequency of snapshots (days).",
						},
						keyRetention: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Retention of snapshots.",
						},
						keyRetentionUnit: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Retention unit.",
						},
					},
				},
				Computed:    true,
				Description: "Daily schedule of the SLA Domain.",
			},
			keyHourlySchedule: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyFrequency: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Frequency of snapshots (hours).",
						},
						keyRetention: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Retention.",
						},
						keyRetentionUnit: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Retention unit.",
						},
					},
				},
				Computed:    true,
				Description: "Hourly schedule.",
			},
			keyMinuteSchedule: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyFrequency: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Frequency (minutes).",
						},
						keyRetention: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Retention.",
						},
						keyRetentionUnit: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Retention unit.",
						},
					},
				},
				Computed:    true,
				Description: "Minute schedule.",
			},
			keyMonthlySchedule: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyFrequency: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Frequency (months).",
						},
						keyRetention: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Retention.",
						},
						keyRetentionUnit: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Retention unit.",
						},
						keyDayOfMonth: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Day of month.",
						},
					},
				},
				Computed:    true,
				Description: "Monthly schedule.",
			},
			keyQuarterlySchedule: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyFrequency: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Frequency (quarters).",
						},
						keyRetention: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Retention.",
						},
						keyRetentionUnit: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Retention unit.",
						},
						keyDayOfQuarter: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Day of quarter.",
						},
						keyQuarterStartMonth: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Quarter start month.",
						},
					},
				},
				Computed:    true,
				Description: "Quarterly schedule.",
			},
			keyWeeklySchedule: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyFrequency: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Frequency (weeks).",
						},
						keyRetention: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Retention.",
						},
						keyRetentionUnit: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Retention unit.",
						},
						keyDayOfWeek: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Day of week.",
						},
					},
				},
				Computed:    true,
				Description: "Weekly schedule.",
			},
			keyYearlySchedule: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyFrequency: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Frequency (years).",
						},
						keyRetention: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Retention.",
						},
						keyRetentionUnit: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Retention unit.",
						},
						keyDayOfYear: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Day of year.",
						},
						keyYearStartMonth: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Year start month.",
						},
					},
				},
				Computed:    true,
				Description: "Yearly schedule.",
			},
		},
	}
}

func slaDomainRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "slaDomainRead")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	var slaDomain gqlsla.Domain
	if id := d.Get(keyID).(string); id != "" {
		id, err := uuid.Parse(id)
		if err != nil {
			return diag.FromErr(err)
		}
		slaDomain, err = sla.Wrap(client).DomainByID(ctx, id)
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		slaDomain, err = sla.Wrap(client).DomainByName(ctx, d.Get(keyName).(string))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set(keyName, slaDomain.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyDescription, slaDomain.Description); err != nil {
		return diag.FromErr(err)
	}

	objectTypes := &schema.Set{F: schema.HashString}
	for _, objectType := range slaDomain.ObjectTypes {
		objectTypes.Add(string(objectType))
	}
	if err := d.Set(keyObjectTypes, objectTypes); err != nil {
		return diag.FromErr(err)
	}

	if archivalSpecs := toArchivalSpecs(slaDomain); archivalSpecs != nil {
		if err := d.Set(keyArchivalSpec, archivalSpecs); err != nil {
			return diag.FromErr(err)
		}
	}

	if daily := toDailySchedule(slaDomain); daily != nil {
		if err := d.Set(keyDailySchedule, daily); err != nil {
			return diag.FromErr(err)
		}
	}
	if hourly := toHourlySchedule(slaDomain); hourly != nil {
		if err := d.Set(keyHourlySchedule, hourly); err != nil {
			return diag.FromErr(err)
		}
	}
	if minutely := toMinuteSchedule(slaDomain); minutely != nil {
		if err := d.Set(keyMinuteSchedule, minutely); err != nil {
			return diag.FromErr(err)
		}
	}
	if monthly := toMonthlySchedule(slaDomain); monthly != nil {
		if err := d.Set(keyMonthlySchedule, monthly); err != nil {
			return diag.FromErr(err)
		}
	}
	if quarterly := toQuarterlySchedule(slaDomain); quarterly != nil {
		if err := d.Set(keyQuarterlySchedule, quarterly); err != nil {
			return diag.FromErr(err)
		}
	}
	if weekly := toWeeklySchedule(slaDomain); weekly != nil {
		if err := d.Set(keyWeeklySchedule, weekly); err != nil {
			return diag.FromErr(err)
		}
	}
	if yearly := toYearlySchedule(slaDomain); yearly != nil {
		if err := d.Set(keyYearlySchedule, yearly); err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(slaDomain.ID.String())
	return nil
}

func archivalSpecResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			keyFrequencies: {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "",
			},
			keyThreshold: {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "",
			},
			keyThresholdUnit: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "",
			},
		},
	}
}

func toArchivalSpecs(slaDomain gqlsla.Domain) *schema.Set {
	if len(slaDomain.ArchivalSpecs) == 0 {
		return nil
	}

	archivalSpecs := &schema.Set{F: schema.HashResource(archivalSpecResource())}
	for _, archivalSpec := range slaDomain.ArchivalSpecs {
		frequencies := &schema.Set{F: schema.HashString}
		for _, freq := range archivalSpec.Frequencies {
			frequencies.Add(string(freq))
		}

		archivalSpecs.Add(map[string]interface{}{
			keyFrequencies:   frequencies,
			keyThreshold:     archivalSpec.Threshold,
			keyThresholdUnit: string(archivalSpec.ThresholdUnit),
		})
	}

	return archivalSpecs
}
