package esxi

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

// buildGuestResourceSchema builds the guest resource schema
func buildGuestResourceSchema() *schema.Resource {
	return &schema.Resource{
		Create: createGuestResource,
		Read:   readGuestDataIntoResource,
		Update: updateGuestResource,
		Delete: deleteGuestResource,
		Importer: &schema.ResourceImporter{
			State: importGuestResource,
		},
		Schema: map[string]*schema.Schema{
			"count": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    true,
				Default:     1,
				Description: "Number of VM instances to create.",
			},
			"clone_from_vm": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				DefaultFunc: schema.EnvDefaultFunc("clone_from_vm", nil),
				Description: "Source vm path on esxi host to clone.",
			},
			"ovf_source": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				DefaultFunc: schema.EnvDefaultFunc("ovf_source", nil),
				Description: "Local path to source ovf files.",
			},
			"disk_store": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: schema.EnvDefaultFunc("disk_store", "Least Used"),
				Description: "esxi diskstore for boot disk.",
			},
			"resource_pool_name": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Computed:    true,
				Description: "Resource pool name to place guest.",
			},
			"guest_name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: schema.EnvDefaultFunc("guest_name", "vm-example"),
				Description: "esxi guest name.",
			},
			"boot_disk_type": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Guest boot disk type. thin, zeroedthick, eagerzeroedthick",
			},
			"boot_disk_size": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
				Computed:    true,
				DefaultFunc: schema.EnvDefaultFunc("boot_disk_size", nil),
				Description: "Guest boot disk size. Will expand boot disk to this size.",
			},
			"memsize": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
				Computed:    true,
				Description: "Guest guest memory size.",
			},
			"numvcpus": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
				Computed:    true,
				Description: "Guest guest number of virtual cpus.",
			},
			"virthwver": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
				Computed:    true,
				Description: "Guest Virtual HW version.",
			},
			"guestos": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
				Computed:    true,
				Description: "Guest OS type.",
			},
			"network_interfaces": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: false,
				Default:  nil,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"virtual_network": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: false,
							Computed: true,
						},
						"mac_address": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: false,
							Computed: true,
						},
						"nic_type": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: false,
							Computed: true,
						},
						"pci_slot_number": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: false,
							Computed: true,
						},
					},
				},
			},
			"power": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
				Computed:    true,
				Description: "Guest power state.",
				DefaultFunc: schema.EnvDefaultFunc("power", "on"),
			},
			//  Calculated only, you cannot overwrite this.
			"ip_address": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The IP address reported by VMware tools.",
			},
			"guest_startup_timeout": {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				Description:  "The amount of guest uptime, in seconds, to wait for an available IP address on this virtual machine.",
				ValidateFunc: validation.IntBetween(1, 600),
			},
			"guest_shutdown_timeout": {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				Description:  "The amount of time, in seconds, to wait for a graceful shutdown before doing a forced power off.",
				ValidateFunc: validation.IntBetween(0, 600),
			},
			"virtual_disks": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Computed: false,
				Default:  nil,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"virtual_disk_id": &schema.Schema{
							Type:        schema.TypeString,
							Required:    true,
							DefaultFunc: schema.EnvDefaultFunc("virtual_disk_id", ""),
						},
						"slot": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: "SCSI_Ctrl:SCSI_id.    Range  '0:1' to '0:15'.   SCSI_id 7 is not allowed.",
						},
					},
				},
			},
			"notes": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
				Computed:    true,
				Description: "Guest notes (annotation).",
			},
			"guestinfo": &schema.Schema{
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "pass data to VM",
				ForceNew:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}
