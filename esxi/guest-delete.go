package esxi

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
)

// DeleteGuestResource deletes the guest resource
func DeleteGuestResource(d *schema.ResourceData, m interface{}) error {
	c := m.(*Config)
	esxiSSHinfo := SSHConnectionSettings{c.esxiHostName, c.esxiHostPort, c.esxiUserName, c.esxiPassword}
	log.Println("[resourceGUESTDelete]")

	var remoteCmd, stdout string
	var err error

	vmid := d.Id()
	guestShutdownTimeout := d.Get("guest_shutdown_timeout").(int)

	_, err = PowerOffGuest(c, vmid, guestShutdownTimeout)
	if err != nil {
		return err
	}

	// remove storage from vmx so it doesn't get deleted by the vim-cmd destroy
	err = CleanVmxStorage(c, vmid)
	if err != nil {
		log.Printf("[resourceGUESTDelete] Failed clean storage from vmid: %s (to be deleted)\n", vmid)
	}

	time.Sleep(5 * time.Second)
	remoteCmd = fmt.Sprintf("vim-cmd vmsvc/destroy %s", vmid)
	stdout, err = RunHostCommand(esxiSSHinfo, remoteCmd, "vmsvc/destroy")
	if err != nil {
		// todo more descriptive err message
		log.Printf("[resourceGUESTDelete] Failed destroy vmid: %s\n", stdout)
		return err
	}

	d.SetId("")

	return nil
}
