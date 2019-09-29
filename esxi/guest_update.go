package esxi

import (
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
)

// UpdateGuestResource updates the guest resource
func UpdateGuestResource(d *schema.ResourceData, m interface{}) error {
	c := m.(*Config)
	log.Printf("[resourceGUESTUpdate]\n")

	var virtualNetworks [10][3]string
	var virtualDisks [60][2]string
	var i int
	var err error

	vmid := d.Id()
	memsize := d.Get("memsize").(string)
	numvcpus := d.Get("numvcpus").(string)
	bootDiskSize := d.Get("boot_disk_size").(string)
	virthwver := d.Get("virthwver").(string)
	guestos := d.Get("guestos").(string)
	guestShutdownTimeout := d.Get("guest_shutdown_timeout").(int)
	notes := d.Get("notes").(string)
	lanAdaptersCount := d.Get("network_interfaces.#").(int)
	power := d.Get("power").(string)

	guestinfo, ok := d.Get("guestinfo").(map[string]interface{})
	if !ok {
		return errors.New("guestinfo is wrong type")
	}

	if lanAdaptersCount > 10 {
		lanAdaptersCount = 10
	}
	for i := 0; i < lanAdaptersCount; i++ {
		prefix := fmt.Sprintf("network_interfaces.%d.", i)

		if attr, ok := d.Get(prefix + "virtual_network").(string); ok && attr != "" {
			virtualNetworks[i][0] = d.Get(prefix + "virtual_network").(string)
		}
		if attr, ok := d.Get(prefix + "mac_address").(string); ok && attr != "" {
			virtualNetworks[i][1] = d.Get(prefix + "mac_address").(string)
		}
		if attr, ok := d.Get(prefix + "nic_type").(string); ok && attr != "" {
			virtualNetworks[i][2] = d.Get(prefix + "nic_type").(string)
		}
	}

	//  Validate virtual_disks
	virtualDiskCount := d.Get("virtual_disks.#").(int)
	if virtualDiskCount > 59 {
		virtualDiskCount = 59
	}

	// Validate guestOS
	if validateGuestOsType(guestos) == false {
		return errors.New("Error: invalid guestos.  see https://github.com/josenk/vagrant-vmware-esxi/wiki/VMware-ESXi-6.5-guestOS-types")
	}

	for i = 0; i < virtualDiskCount; i++ {
		prefix := fmt.Sprintf("virtual_disks.%d.", i)

		if attr, ok := d.Get(prefix + "virtual_disk_id").(string); ok && attr != "" {
			virtualDisks[i][0] = d.Get(prefix + "virtual_disk_id").(string)
		}

		if attr, ok := d.Get(prefix + "slot").(string); ok && attr != "" {
			// todo validate slots are in format "0-3:0-15"
			virtualDisks[i][1] = d.Get(prefix + "slot").(string)
		}
	}

	//
	//   Power off guest if it's powered on.
	//
	currentpowerstate := GetGuestPowerState(c, vmid)
	if currentpowerstate == "on" || currentpowerstate == "suspended" {
		_, err = PowerOffGuest(c, vmid, guestShutdownTimeout)
		if err != nil {
			return err
		}
	}

	//
	//  make updates to vmx file
	//
	imemsize, _ := strconv.Atoi(memsize)
	inumvcpus, _ := strconv.Atoi(numvcpus)
	ivirthwver, _ := strconv.Atoi(virthwver)
	err = UpdateVmx(c, vmid, false, imemsize, inumvcpus, ivirthwver, guestos, virtualNetworks, virtualDisks, notes, guestinfo)
	if err != nil {
		fmt.Println("Failed to update VMX file.")
		return errors.New("Failed to update VMX file")
	}

	//
	//  Grow boot disk to boot_disk_size
	//
	bootDiskPath, _ := GetBootDiskPath(c, vmid)

	err = GrowVirtualDisk(c, bootDiskPath, bootDiskSize)
	if err != nil {
		return errors.New("Failed to grow boot disk")
	}

	//  power on
	if power == "on" {
		_, err = PowerOnGuest(c, vmid)
		if err != nil {
			fmt.Println("Failed to power on.")
			return errors.New("Failed to power on")
		}
	}

	return ReadGuestDataIntoResource(d, m)
}
