// Copyright IBM Corp. 2017, 2021 All Rights Reserved.
// Licensed under the Mozilla Public License v2.0

package power

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	st "github.com/IBM-Cloud/power-go-client/clients/instance"
	"github.com/IBM-Cloud/power-go-client/power/models"
	"github.com/IBM-Cloud/terraform-provider-ibm/ibm/conns"
	"github.com/IBM-Cloud/terraform-provider-ibm/ibm/flex"
	"github.com/IBM-Cloud/terraform-provider-ibm/ibm/validate"
)

// Attributes and Arguments defined in data_source_ibm_pi_network.go
func ResourceIBMPINetwork() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceIBMPINetworkCreate,
		ReadContext:   resourceIBMPINetworkRead,
		UpdateContext: resourceIBMPINetworkUpdate,
		DeleteContext: resourceIBMPINetworkDelete,
		Importer:      &schema.ResourceImporter{},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(60 * time.Minute),
			Update: schema.DefaultTimeout(60 * time.Minute),
			Delete: schema.DefaultTimeout(60 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			PINetworkType: {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validate.ValidateAllowedStringValues([]string{"vlan", "pub-vlan"}),
				Description:  "PI network type",
			},
			PINetworkName: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "PI network name",
			},
			PINetworkDNS: {
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "List of PI network DNS name",
			},
			PINetworkCIDR: {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "PI network CIDR",
			},
			// helpers.PINetworkGateway: {
			// 	Type:        schema.TypeString,
			// 	Optional:    true,
			// 	Description: "PI network gateway",
			// },
			PINetworkJumbo: {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "PI network enable MTU Jumbo option",
			},
			PICloudInstanceID: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "PI cloud instance ID",
			},

			//Computed Attributes
			NetworkNetworkID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "PI network ID",
			},
			NetworkVlanID: {
				Type:        schema.TypeFloat,
				Computed:    true,
				Description: "VLAN Id value",
			},
		},
	}
}

func resourceIBMPINetworkCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sess, err := meta.(conns.ClientSession).IBMPISession()
	if err != nil {
		return diag.FromErr(err)
	}
	cloudInstanceID := d.Get(PICloudInstanceID).(string)
	networkname := d.Get(PINetworkName).(string)
	networktype := d.Get(PINetworkType).(string)
	networkdns := flex.ExpandStringList((d.Get(PINetworkDNS).(*schema.Set)).List())

	client := st.NewIBMPINetworkClient(ctx, sess, cloudInstanceID)
	var body = &models.NetworkCreate{
		Type: &networktype,
		Name: networkname,
	}
	if v, ok := d.GetOk(PINetworkJumbo); ok {
		body.Jumbo = v.(bool)
	}
	if len(networkdns) > 0 {
		body.DNSServers = networkdns
	}

	if networktype == "vlan" {
		var networkcidr string
		if v, ok := d.GetOk(PINetworkCIDR); ok {
			networkcidr = v.(string)
		} else {
			diag.Errorf("%s is required when %s is vlan", PINetworkCIDR, PINetworkType)
		}

		gateway, firstip, lastip, err := generateIPData(networkcidr)
		if err != nil {
			return diag.FromErr(err)
		}

		var ipbody = []*models.IPAddressRange{{EndingIPAddress: &lastip, StartingIPAddress: &firstip}}
		body.IPAddressRanges = ipbody
		body.Gateway = gateway
		body.Cidr = networkcidr
	}

	networkResponse, err := client.Create(body)
	if err != nil {
		return diag.FromErr(err)
	}

	IBMPINetworkID := *networkResponse.NetworkID

	d.SetId(fmt.Sprintf("%s/%s", cloudInstanceID, IBMPINetworkID))

	_, err = isWaitForIBMPINetworkAvailable(ctx, client, IBMPINetworkID, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceIBMPINetworkRead(ctx, d, meta)
}

func resourceIBMPINetworkRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sess, err := meta.(conns.ClientSession).IBMPISession()
	if err != nil {
		return diag.FromErr(err)
	}

	cloudInstanceID, networkID, err := splitID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	networkC := st.NewIBMPINetworkClient(ctx, sess, cloudInstanceID)
	networkdata, err := networkC.Get(networkID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set(NetworkNetworkID, networkdata.NetworkID)
	d.Set(PINetworkCIDR, networkdata.Cidr)
	d.Set(PINetworkDNS, networkdata.DNSServers)
	d.Set(NetworkVlanID, networkdata.VlanID)
	d.Set(PINetworkName, networkdata.Name)
	d.Set(PINetworkType, networkdata.Type)
	d.Set(PINetworkJumbo, networkdata.Jumbo)

	return nil

}

func resourceIBMPINetworkUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sess, err := meta.(conns.ClientSession).IBMPISession()
	if err != nil {
		return diag.FromErr(err)
	}

	cloudInstanceID, networkID, err := splitID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChanges(PINetworkName, PINetworkDNS) {
		networkC := st.NewIBMPINetworkClient(ctx, sess, cloudInstanceID)
		body := &models.NetworkUpdate{
			DNSServers: flex.ExpandStringList((d.Get(PINetworkDNS).(*schema.Set)).List()),
		}

		if d.HasChange(PINetworkName) {
			name := d.Get(PINetworkName).(string)
			body.Name = &name
		}

		_, err = networkC.Update(networkID, body)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceIBMPINetworkRead(ctx, d, meta)
}

func resourceIBMPINetworkDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	log.Printf("Calling the network delete functions. ")
	sess, err := meta.(conns.ClientSession).IBMPISession()
	if err != nil {
		return diag.FromErr(err)
	}

	cloudInstanceID, networkID, err := splitID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	networkC := st.NewIBMPINetworkClient(ctx, sess, cloudInstanceID)
	err = networkC.Delete(networkID)

	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")
	return nil
}

func isWaitForIBMPINetworkAvailable(ctx context.Context, client *st.IBMPINetworkClient, id string, timeout time.Duration) (interface{}, error) {
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"retry", "build"},
		Target:     []string{"NETWORK_READY"},
		Refresh:    isIBMPINetworkRefreshFunc(client, id),
		Timeout:    timeout,
		Delay:      10 * time.Second,
		MinTimeout: 10 * time.Second,
	}

	return stateConf.WaitForStateContext(ctx)
}

func isIBMPINetworkRefreshFunc(client *st.IBMPINetworkClient, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		network, err := client.Get(id)
		if err != nil {
			return nil, "", err
		}

		if network.VlanID != nil {
			return network, "NETWORK_READY", nil
		}

		return network, "build", nil
	}
}

func generateIPData(cdir string) (gway, firstip, lastip string, err error) {
	_, ipv4Net, err := net.ParseCIDR(cdir)

	if err != nil {
		return "", "", "", err
	}

	var subnetToSize = map[string]int{
		"21": 2048,
		"22": 1024,
		"23": 512,
		"24": 256,
		"25": 128,
		"26": 64,
		"27": 32,
		"28": 16,
		"29": 8,
		"30": 4,
		"31": 2,
	}

	//subnetsize, _ := ipv4Net.Mask.Size()

	gateway, err := cidr.Host(ipv4Net, 1)
	if err != nil {
		log.Printf("Failed to get the gateway for this cidr passed in %s", cdir)
		return "", "", "", err
	}
	ad := cidr.AddressCount(ipv4Net)

	convertedad := strconv.FormatUint(ad, 10)
	// Powervc in wdc04 has to reserve 3 ip address hence we start from the 4th. This will be the default behaviour
	firstusable, err := cidr.Host(ipv4Net, 4)
	if err != nil {
		log.Print(err)
		return "", "", "", err
	}
	lastusable, err := cidr.Host(ipv4Net, subnetToSize[convertedad]-2)
	if err != nil {
		log.Print(err)
		return "", "", "", err
	}
	return gateway.String(), firstusable.String(), lastusable.String(), nil

}
