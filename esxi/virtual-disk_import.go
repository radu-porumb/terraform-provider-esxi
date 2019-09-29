package esxi

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
)

// ImportVirtualDiskDataIntoResource imports virtual disk data from ESXi host into resource
func ImportVirtualDiskDataIntoResource(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	c := m.(*Config)
	log.Println("[resourceVIRTUALDISKImport]")

	results := make([]*schema.ResourceData, 1, 1)
	results[0] = d

	_, _, _, _, _, err := ReadVirtualDiskInfo(c, d.Id())
	if err != nil {
		d.SetId("")
		return results, fmt.Errorf("Failed to validate virtual_disk: %s", err)
	}

	d.SetId(d.Id())

	return results, nil
}
