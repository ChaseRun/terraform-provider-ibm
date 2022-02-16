// Copyright IBM Corp. 2017, 2021 All Rights Reserved.
// Licensed under the Mozilla Public License v2.0

package power

import (
	//"fmt"
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/IBM-Cloud/power-go-client/clients/instance"
	"github.com/IBM-Cloud/terraform-provider-ibm/ibm/conns"
)

// Attributes and Arguments defined in data_source_ibm_pi_network.go
func DataSourceIBMPIPublicNetwork() *schema.Resource {

	return &schema.Resource{
		ReadContext: dataSourceIBMPIPublicNetworkRead,
		Schema: map[string]*schema.Schema{
			PICloudInstanceID: {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
			},

			// Computed Attributes
			NetworkName: {
				Type:     schema.TypeString,
				Computed: true,
			},
			NetworkType: {
				Type:     schema.TypeString,
				Computed: true,
			},
			NetworkVlanID: {
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
	}
}

func dataSourceIBMPIPublicNetworkRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sess, err := meta.(conns.ClientSession).IBMPISession()
	if err != nil {
		return diag.FromErr(err)
	}

	cloudInstanceID := d.Get(PICloudInstanceID).(string)

	networkC := instance.NewIBMPINetworkClient(ctx, sess, cloudInstanceID)
	networkdata, err := networkC.GetAllPublic()
	if err != nil {
		return diag.FromErr(err)
	}
	if len(networkdata.Networks) < 1 {
		return diag.Errorf("error getting public network or no public network found in %s", cloudInstanceID)
	}

	d.SetId(*networkdata.Networks[0].NetworkID)
	if networkdata.Networks[0].Type != nil {
		d.Set(NetworkType, networkdata.Networks[0].Type)
	}
	if networkdata.Networks[0].Name != nil {
		d.Set(NetworkName, networkdata.Networks[0].Name)
	}
	if networkdata.Networks[0].VlanID != nil {
		d.Set(NetworkVlanID, networkdata.Networks[0].VlanID)
	}

	return nil
}
