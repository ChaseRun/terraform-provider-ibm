// Copyright IBM Corp. 2017, 2021 All Rights Reserved.
// Licensed under the Mozilla Public License v2.0

package power

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	st "github.com/IBM-Cloud/power-go-client/clients/instance"
	"github.com/IBM-Cloud/power-go-client/power/models"
	"github.com/IBM-Cloud/terraform-provider-ibm/ibm/conns"
	"github.com/IBM-Cloud/terraform-provider-ibm/ibm/flex"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Attributes and Arguments defined in data_source_ibm_pi_network_port.go
func ResourceIBMPINetworkPortAttach() *schema.Resource {
	return &schema.Resource{

		CreateContext: resourceIBMPINetworkPortAttachCreate,
		ReadContext:   resourceIBMPINetworkPortAttachRead,
		UpdateContext: resourceIBMPINetworkPortAttachUpdate,
		DeleteContext: resourceIBMPINetworkPortAttachDelete,
		Importer:      &schema.ResourceImporter{},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(60 * time.Minute),
			Delete: schema.DefaultTimeout(60 * time.Minute),
			Update: schema.DefaultTimeout(60 * time.Minute),
		},
		Schema: map[string]*schema.Schema{
			PINetworkPortID: {
				Type:     schema.TypeString,
				Required: true,
			},
			PICloudInstanceID: {
				Type:     schema.TypeString,
				Required: true,
			},
			PINetworkPortInstanceName: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Instance name to attach the network port to",
			},
			PINetworkPortName: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Network Name - This is the subnet name  in the Cloud instance",
			},
			PINetworkPortDescription: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A human readable description for this network Port",
				Default:     "Port Created via Terraform",
			},

			// Attributes
			NetworkPortPublicIP: {
				Type:     schema.TypeString,
				Computed: true,
			},
			NetworkPortIP: {
				Type:     schema.TypeString,
				Computed: true,
			},
			NetworkPortMAC: {
				Type:     schema.TypeString,
				Computed: true,
			},
			NetworkPortStatus: {
				Type:     schema.TypeString,
				Computed: true,
			},
			NetworkPortID: {
				Type:     schema.TypeString,
				Computed: true,
			},
			NetworkPortInstance: {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}

}

func resourceIBMPINetworkPortAttachCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sess, err := meta.(conns.ClientSession).IBMPISession()
	if err != nil {
		return diag.FromErr(err)
	}

	cloudInstanceID := d.Get(PICloudInstanceID).(string)
	networkname := d.Get(PINetworkPortName).(string)
	portid := d.Get(PINetworkPortID).(string)
	instancename := d.Get(PINetworkPortInstanceName).(string)
	description := d.Get(PINetworkPortDescription).(string)
	client := st.NewIBMPINetworkClient(ctx, sess, cloudInstanceID)

	log.Printf("Printing the input to the resource: cloud instance [%s] and network name [%s] and the portid [%s]", cloudInstanceID, networkname, portid)
	body := &models.NetworkPortUpdate{
		Description:   &description,
		PvmInstanceID: &instancename,
	}
	networkPortResponse, err := client.UpdatePort(networkname, portid, body)

	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("Printing the networkresponse %+v", &networkPortResponse)

	IBMPINetworkPortID := *networkPortResponse.PortID

	d.SetId(fmt.Sprintf("%s/%s/%s", cloudInstanceID, networkname, IBMPINetworkPortID))

	_, err = isWaitForIBMPINetworkPortAttachAvailable(ctx, client, IBMPINetworkPortID, networkname, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceIBMPINetworkPortAttachRead(ctx, d, meta)
}

func resourceIBMPINetworkPortAttachRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("Calling ther Network Port Attach Read code")
	sess, err := meta.(conns.ClientSession).IBMPISession()

	if err != nil {
		fmt.Printf("failed to get  a session from the IBM Cloud Service %v", err)
	}

	parts, err := flex.IdParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	cloudInstanceID := parts[0]
	networkID := parts[1]
	portID := parts[2]

	networkC := st.NewIBMPINetworkClient(ctx, sess, cloudInstanceID)
	networkdata, err := networkC.GetPort(networkID, portID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set(NetworkPortIP, networkdata.IPAddress)
	d.Set(NetworkPortMAC, networkdata.MacAddress)
	d.Set(NetworkPortStatus, networkdata.Status)
	d.Set(NetworkPortID, networkdata.PortID)
	d.Set(NetworkPortInstance, networkdata.PvmInstance.Href)
	d.Set(NetworkPortPublicIP, networkdata.ExternalIP)

	return nil
}

func resourceIBMPINetworkPortAttachUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("Calling the attach update ")
	return nil
}

func resourceIBMPINetworkPortAttachDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("Detaching the network port from the Instance ")

	sess, err := meta.(conns.ClientSession).IBMPISession()
	if err != nil {
		fmt.Printf("failed to get  a session from the IBM Cloud Service %v", err)

	}

	parts, err := flex.IdParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	cloudInstanceID := parts[0]
	networkID := parts[1]
	portID := parts[2]

	client := st.NewIBMPINetworkClient(ctx, sess, cloudInstanceID)
	log.Printf("Executing network port detach")
	emptyPVM := ""
	body := &models.NetworkPortUpdate{
		PvmInstanceID: &emptyPVM,
	}
	networkPort, err := client.UpdatePort(networkID, portID, body)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("Printing the networkresponse %+v", &networkPort)

	d.SetId("")
	return nil

}

func isWaitForIBMPINetworkPortAttachAvailable(ctx context.Context, client *st.IBMPINetworkClient, id string, networkname string, timeout time.Duration) (interface{}, error) {
	log.Printf("Waiting for Power Network (%s) that was created for Network Zone (%s) to be available.", id, networkname)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"retry", "build"},
		Target:     []string{"ACTIVE"},
		Refresh:    isIBMPINetworkPortAttachRefreshFunc(client, id, networkname),
		Timeout:    timeout,
		Delay:      10 * time.Second,
		MinTimeout: 10 * time.Minute,
	}

	return stateConf.WaitForStateContext(ctx)
}

func isIBMPINetworkPortAttachRefreshFunc(client *st.IBMPINetworkClient, id, networkname string) resource.StateRefreshFunc {

	log.Printf("Calling the IsIBMPINetwork Refresh Function....with the following id (%s) for network port and following id (%s) for network name and waiting for network to be READY", id, networkname)
	return func() (interface{}, string, error) {
		network, err := client.GetPort(networkname, id)
		if err != nil {
			return nil, "", err
		}

		if &network.PortID != nil && &network.PvmInstance.PvmInstanceID != nil {
			//if network.State == "available" {
			log.Printf(" The port has been created with the following ip address and attached to an instance ")
			return network, "ACTIVE", nil
		}

		return network, "build", nil
	}
}
