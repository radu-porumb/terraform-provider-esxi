package esxi

import (
	"log"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceRESOURCEPOOLRead(d *schema.ResourceData, m interface{}) error {
	c := m.(*Config)

	log.Println("[resourceRESOURCEPOOLRead]")

	var cpuShares, memShares string
	var cpuMin, cpuMax, memMin, memMax int
	var resourcePoolName, cpuMinExpandable, memMinExpandable string
	var err error

	poolID := d.Id()

	// Refresh
	resourcePoolName, cpuMin, cpuMinExpandable, cpuMax, cpuShares, memMin, memMinExpandable, memMax, memShares, err = readResourcePoolData(c, poolID)
	if err != nil {
		d.SetId("")
		return nil
	}

	d.Set("resource_pool_name", resourcePoolName)
	d.Set("cpu_min", cpuMin)
	d.Set("cpu_min_expandable", cpuMinExpandable)
	d.Set("cpu_max", cpuMax)
	d.Set("cpu_shares", cpuShares)
	d.Set("mem_min", memMin)
	d.Set("mem_min_expandable", memMinExpandable)
	d.Set("mem_max", memMax)
	d.Set("mem_shares", memShares)

	return nil
}
