package esxi

import (
	"errors"
	"log"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
)

// updateVirtualDisk updates the virtual disk on the host using the size specified in the resource
func updateVirtualDisk(d *schema.ResourceData, m interface{}) error {
	c := m.(*Config)

	log.Println("[resourceVIRTUALDISKUpdate]")

	if d.HasChange("virtual_disk_size") {
		_, _, _, currentVirtDiskSize, _, err := readVirtualDiskInfo(c, d.Id())
		if err != nil {
			d.SetId("")
			return err
		}

		virtDiskSize := d.Get("virtual_disk_size").(int)

		if currentVirtDiskSize > virtDiskSize {
			return errors.New("Not able to shrink virtual disk:" + d.Id())
		}

		err = growVirtualDisk(c, d.Id(), strconv.Itoa(virtDiskSize))
		if err != nil {
			return errors.New("Failed to grow disk:" + d.Id())
		}
	}

	return nil
}
