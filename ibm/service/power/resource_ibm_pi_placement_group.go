// Copyright IBM Corp. 2021 All Rights Reserved.
// Licensed under the Mozilla Public License v2.0

package power

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	st "github.com/IBM-Cloud/power-go-client/clients/instance"
	models "github.com/IBM-Cloud/power-go-client/power/models"
	"github.com/IBM-Cloud/terraform-provider-ibm/ibm/conns"
	"github.com/IBM-Cloud/terraform-provider-ibm/ibm/flex"
	"github.com/IBM-Cloud/terraform-provider-ibm/ibm/validate"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

// Attributes and Arguments defined in data_source_ibm_pi_placement_group.go
func ResourceIBMPIPlacementGroup() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceIBMPIPlacementGroupCreate,
		ReadContext:   resourceIBMPIPlacementGroupRead,
		UpdateContext: resourceIBMPIPlacementGroupUpdate,
		DeleteContext: resourceIBMPIPlacementGroupDelete,
		Importer:      &schema.ResourceImporter{},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(60 * time.Minute),
			Update: schema.DefaultTimeout(60 * time.Minute),
			Delete: schema.DefaultTimeout(60 * time.Minute),
		},

		Schema: map[string]*schema.Schema{

			PIPlacementGroupName: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the placement group",
			},

			PIPlacementGroupPolicy: {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validate.ValidateAllowedStringValues([]string{"affinity", "anti-affinity"}),
				Description:  "Policy of the placement group",
			},

			PICloudInstanceID: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "PI cloud instance ID",
			},

			PlacementGroupMembers: {
				Type:        schema.TypeSet,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Server IDs that are the placement group members",
			},

			PlacementGroupID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "PI placement group ID",
			},
		},
	}
}

func resourceIBMPIPlacementGroupCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sess, err := meta.(conns.ClientSession).IBMPISession()
	if err != nil {
		return diag.FromErr(err)
	}

	cloudInstanceID := d.Get(PICloudInstanceID).(string)
	name := d.Get(PIPlacementGroupName).(string)
	policy := d.Get(PIPlacementGroupPolicy).(string)
	client := st.NewIBMPIPlacementGroupClient(ctx, sess, cloudInstanceID)
	body := &models.PlacementGroupCreate{
		Name:   &name,
		Policy: &policy,
	}

	response, err := client.Create(body)
	if err != nil {
		log.Printf("[DEBUG]  err %s", err)
		return diag.FromErr(err)
	}

	log.Printf("Printing the placement group %+v", &response)

	d.SetId(fmt.Sprintf("%s/%s", cloudInstanceID, *response.ID))
	return resourceIBMPIPlacementGroupRead(ctx, d, meta)
}

func resourceIBMPIPlacementGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	sess, err := meta.(conns.ClientSession).IBMPISession()
	if err != nil {
		return diag.FromErr(err)
	}
	parts, err := flex.IdParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	cloudInstanceID := parts[0]
	client := st.NewIBMPIPlacementGroupClient(ctx, sess, cloudInstanceID)

	response, err := client.Get(parts[1])
	if err != nil {
		log.Printf("[DEBUG]  err %s", err)
		return diag.FromErr(err)
	}

	d.Set(PIPlacementGroupName, response.Name)
	d.Set(PlacementGroupID, response.ID)
	d.Set(PIPlacementGroupPolicy, response.Policy)
	d.Set(PlacementGroupMembers, response.Members)

	return nil

}

func resourceIBMPIPlacementGroupUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return resourceIBMPIPlacementGroupRead(ctx, d, meta)
}

func resourceIBMPIPlacementGroupDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sess, err := meta.(conns.ClientSession).IBMPISession()
	if err != nil {
		return diag.FromErr(err)
	}
	parts, err := flex.IdParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	cloudInstanceID := parts[0]
	client := st.NewIBMPIPlacementGroupClient(ctx, sess, cloudInstanceID)
	err = client.Delete(parts[1])

	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")
	return nil
}
