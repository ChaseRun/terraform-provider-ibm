// Copyright IBM Corp. 2017, 2021 All Rights Reserved.
// Licensed under the Mozilla Public License v2.0

package power

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/IBM-Cloud/power-go-client/clients/instance"
	"github.com/IBM-Cloud/power-go-client/power/models"
	"github.com/IBM-Cloud/terraform-provider-ibm/ibm/conns"
)

// Attributes and Arguments defined in data_source_ibm_pi_console_languages.go
func ResourceIBMPIInstanceConsoleLanguage() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceIBMPIInstanceConsoleLanguageCreate,
		ReadContext:   resourceIBMPIInstanceConsoleLanguageRead,
		UpdateContext: resourceIBMPIInstanceConsoleLanguageUpdate,
		DeleteContext: resourceIBMPIInstanceConsoleLanguageDelete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(5 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			// Required Attributes
			PICloudInstanceID: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "PI cloud instance ID",
			},
			PIConsoleLanguageName: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The unique identifier or name of the instance",
			},
			ConsoleLanguageCode: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Language code",
			},
		},
	}
}

func resourceIBMPIInstanceConsoleLanguageCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sess, err := meta.(conns.ClientSession).IBMPISession()
	if err != nil {
		return diag.FromErr(err)
	}

	cloudInstanceID := d.Get(PICloudInstanceID).(string)
	instanceName := d.Get(PIConsoleLanguageName).(string)
	code := d.Get(ConsoleLanguageCode).(string)

	client := instance.NewIBMPIInstanceClient(ctx, sess, cloudInstanceID)

	consoleLanguage := &models.ConsoleLanguage{
		Code: &code,
	}

	_, err = client.UpdateConsoleLanguage(instanceName, consoleLanguage)
	if err != nil {
		log.Printf("[DEBUG] err %s", err)
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%s/%s", cloudInstanceID, instanceName))

	return resourceIBMPIInstanceConsoleLanguageRead(ctx, d, meta)
}

func resourceIBMPIInstanceConsoleLanguageRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// There is no get concept for instance console language
	return nil
}

func resourceIBMPIInstanceConsoleLanguageUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sess, err := meta.(conns.ClientSession).IBMPISession()
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange(ConsoleLanguageCode) {
		cloudInstanceID := d.Get(PICloudInstanceID).(string)
		instanceName := d.Get(PIConsoleLanguageName).(string)
		code := d.Get(ConsoleLanguageCode).(string)

		client := instance.NewIBMPIInstanceClient(ctx, sess, cloudInstanceID)

		consoleLanguage := &models.ConsoleLanguage{
			Code: &code,
		}
		_, err = client.UpdateConsoleLanguage(instanceName, consoleLanguage)
		if err != nil {
			log.Printf("[DEBUG] err %s", err)
			return diag.FromErr(err)
		}
	}
	return resourceIBMPIInstanceConsoleLanguageRead(ctx, d, meta)
}

func resourceIBMPIInstanceConsoleLanguageDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// There is no delete or unset concept for instance console language
	d.SetId("")
	return nil
}
