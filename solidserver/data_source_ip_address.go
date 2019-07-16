package solidserver

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/url"
	"regexp"

	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceipaddress() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceipaddressRead,

		Schema: map[string]*schema.Schema{
			/* Removed as suggestion ************************
			"id": {
				Type:        schema.TypeInt,
				Description: "The ID of the IP address.",
				Required:    true,
			},
			*************************************************/
			"space": {
				Type:        schema.TypeString,
				Description: "The name of the space of the IP address.",
				Required:    true,
			},
			"subnet": {
				Type:        schema.TypeString,
				Description: "The name of the subnet of the IP address.",
				Computed:    true,
			},
			"pool": {
				Type:        schema.TypeString,
				Description: "The name of the pool of the IP address.",
				Computed:    true,
			},
			"address": {
				Type:        schema.TypeString,
				Description: "The IP address.",
				Required:    true,
			},
			"device": {
				Type:        schema.TypeString,
				Description: "Device Name associated to the IP address (Require a 'Device Manager' license).",
				Computed:    true,
			},
			"name": {
				Type:        schema.TypeString,
				Description: "The short name or FQDN of the IP address.",
				Computed:    true,
			},
			"mac": {
				Type:        schema.TypeString,
				Description: "The MAC Address of the IP address.",
				Computed:    true,
			},

			"class": {
				Type:        schema.TypeString,
				Description: "The class associated to the IP address.",
				Computed:    true,
			},
			"prefix_size": {
				Type:        schema.TypeInt,
				Description: "The prefix_length associated to the IP address.",
				Computed:    true,
			},
			"class_parameters": {
				Type:        schema.TypeMap,
				Description: "The class parameters associated to the IP address.",
				Computed:    true,
			},
		},
	}
}

func dataSourceipaddressRead(d *schema.ResourceData, meta interface{}) error {
	s := meta.(*SOLIDserver)

	parameters := url.Values{}
	// parameters.Add("ip_id", d.Id())
	parameters.Add("WHERE", "site_name='" + d.Get("space").(string) + "' AND hostaddr='" + d.Get("address").(string) + "'")
	
	// Sending the read request
	log.Printf("[DEBUG] SOLIDServer - lookup for IP address with oid: %s\n", d.Id())
	resp, body, err := s.Request("get", "rest/ip_address_list", &parameters)

	if err == nil {
		var buf [](map[string]interface{})
		json.Unmarshal([]byte(body), &buf)

		// Checking the answer
		if resp.StatusCode == 200 && len(buf) > 0 {
			d.SetId(buf[0]["ip_id"].(string))
			d.Set("space", buf[0]["site_name"].(string))
			d.Set("subnet", buf[0]["subnet_name"].(string))
			d.Set("pool", buf[0]["pool_name"].(string))
			d.Set("address", hexiptoip(buf[0]["ip_addr"].(string)))
			d.Set("name", buf[0]["name"].(string))
			d.Set("prefix_size", 32-math.Round(math.Log(buf[0]["subnet_size"].(float64))+2))

			if macIgnore, _ := regexp.MatchString("^EIP:", buf[0]["mac_addr"].(string)); !macIgnore {
				d.Set("mac", buf[0]["mac_addr"].(string))
			} else {
				d.Set("mac", "")
			}

			d.Set("class", buf[0]["ip_class_name"].(string))

			// Updating local class_parameters
			currentClassParameters := d.Get("class_parameters").(map[string]interface{})
			retrievedClassParameters, _ := url.ParseQuery(buf[0]["ip_class_parameters"].(string))
			computedClassParameters := map[string]string{}

			for ck := range currentClassParameters {
				if rv, rvExist := retrievedClassParameters[ck]; rvExist {
					computedClassParameters[ck] = rv[0]
				} else {
					computedClassParameters[ck] = ""
				}
			}

			d.Set("class_parameters", computedClassParameters)

			return nil
		}

		if len(buf) > 0 {
			if errMsg, errExist := buf[0]["errmsg"].(string); errExist {
				// Log the error
				log.Printf("[DEBUG] SOLIDServer - Unable to find IP address: %s (%s)\n", d.Get("name"), errMsg)
			}
		} else {
			// Log the error
			log.Printf("[DEBUG] SOLIDServer - Unable to find IP address (oid): %s\n", d.Id())
		}

		// Do not unset the local ID to avoid inconsistency

		// Reporting a failure
		return fmt.Errorf("SOLIDServer - Unable to find IP address: %s", d.Get("name").(string))
	}

	// Reporting a failure
	return err
}
