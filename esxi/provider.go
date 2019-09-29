package esxi

import (
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func init() {
	// Terraform is already adding the timestamp for us
	log.SetFlags(log.Lshortfile)
	log.SetPrefix(fmt.Sprintf("pid-%d-", os.Getpid()))
}

// Provider returns the provider schema
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"esxi_hostname": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: schema.EnvDefaultFunc("", "esxi"),
				Description: "The esxi hostname or IP address.",
			},
			"esxi_hostport": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("esxi_hostport", "22"),
				Description: "ssh port.",
			},
			"esxi_username": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("esxi_username", "root"),
				Description: "esxi ssh username.",
			},
			"esxi_password": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("esxi_password", "unset"),
				Description: "esxi ssh password.",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"esxi_guest":         BuildGuestResourceSchema(),
			"esxi_resource_pool": BuildResourcePoolResourceSchema(),
			"esxi_virtual_disk":  BuildVirtualDiskResourceSchema(),
		},
		ConfigureFunc: ConfigureProvider,
	}
}

// ConfigureProvider builds the configuration map from provider data
func ConfigureProvider(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		esxiHostName: d.Get("esxi_hostname").(string),
		esxiHostPort: d.Get("esxi_hostport").(string),
		esxiUserName: d.Get("esxi_username").(string),
		esxiPassword: d.Get("esxi_password").(string),
	}

	if err := config.ValidateEsxiCredentials(); err != nil {
		return nil, err
	}

	return &config, nil
}
