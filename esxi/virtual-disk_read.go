package esxi

import (
	"log"

	"github.com/hashicorp/terraform/helper/schema"
)

// ReadVirtualDiskDataIntoResource reads virtual disk data from ESXi host into resource
func ReadVirtualDiskDataIntoResource(d *schema.ResourceData, m interface{}) error {
	c := m.(*Config)
	log.Println("[resourceVIRTUALDISKRead]")

	virtualDiskDiskStore, virtualDiskDir, virtualDiskName, virtualDiskSize, virtualDiskType, err := ReadVirtualDiskInfo(c, d.Id())
	if err != nil {
		d.SetId("")
		return nil
	}

	d.Set("virtual_disk_disk_store", virtualDiskDiskStore)
	d.Set("virtual_disk_dir", virtualDiskDir)
	d.Set("virtual_disk_name", virtualDiskName)
	d.Set("virtual_disk_size", virtualDiskSize)
	if virtualDiskType != "Unknown" {
		d.Set("virtual_disk_type", virtualDiskType)
	}

	return nil
}
