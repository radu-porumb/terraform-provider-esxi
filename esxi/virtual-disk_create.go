package esxi

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
)

// createVirtualDiskResource creates a virtual disk resource
func createVirtualDiskResource(d *schema.ResourceData, m interface{}) error {
	c := m.(*Config)
	log.Println("[resourceVIRTUALDISKCreate]")

	virtualDiskDiskStore := d.Get("virtual_disk_disk_store").(string)
	virtualDiskDir := d.Get("virtual_disk_dir").(string)
	virtualDiskName := d.Get("virtual_disk_name").(string)
	virtualDiskSize := d.Get("virtual_disk_size").(int)
	virtualDiskType := d.Get("virtual_disk_type").(string)

	if virtualDiskName == "" {
		rand.Seed(time.Now().UnixNano())

		const digits = "0123456789ABCDEF"
		name := make([]byte, 10)
		for i := range name {
			name[i] = digits[rand.Intn(len(digits))]
		}

		virtualDiskName = fmt.Sprintf("vdisk_%s.vmdk", name)
	}

	//
	//  Validate virtual_disk_name
	//

	// todo,  check invalid chars (quotes, slash, period, comma)
	// todo,  must end with .vmdk

	virtDiskID, err := createVirtualDisk(c, virtualDiskDiskStore, virtualDiskDir,
		virtualDiskName, virtualDiskSize, virtualDiskType)
	if err == nil {
		d.SetId(virtDiskID)
	} else {
		log.Println("[resourceVIRTUALDISKCreate] Error: " + err.Error())
		d.SetId("")
		return fmt.Errorf("Failed to create virtual Disk: %s\nError: %s", virtualDiskName, err.Error())
	}

	return nil
}
