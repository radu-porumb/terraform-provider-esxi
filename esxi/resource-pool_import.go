package esxi

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceRESOURCEPOOLImport(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	c := m.(*Config)

	log.Println("[resourceRESOURCEPOOLImport]")

	var err error

	results := make([]*schema.ResourceData, 1, 1)
	results[0] = d

	// get VMID (by name)
	_, err = getResourcePoolName(c, d.Id())
	if err != nil {
		return results, fmt.Errorf("Failed to validate resource_pool: %s", err)
	}

	d.SetId(d.Id())

	return results, nil
}
