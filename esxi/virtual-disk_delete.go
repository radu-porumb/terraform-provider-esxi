package esxi

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
)

// DeleteVirtualDiskResource deletes the virtual disk resource
func DeleteVirtualDiskResource(d *schema.ResourceData, m interface{}) error {
	c := m.(*Config)
	esxiSSHinfo := SSHConnectionSettings{c.esxiHostName, c.esxiHostPort, c.esxiUserName, c.esxiPassword}
	log.Println("[resourceVIRTUALDISKDelete]")

	var remoteCmd, stdout string
	var err error

	virtualDiskID := d.Id()
	virtualDiskDiskStore := d.Get("virtual_disk_disk_store").(string)
	virtualDiskDir := d.Get("virtual_disk_dir").(string)

	//  Destroy virtual disk.
	remoteCmd = fmt.Sprintf("/bin/vmkfstools -U %s", virtualDiskID)
	stdout, err = RunHostCommand(esxiSSHinfo, remoteCmd, "destroy virtual disk")
	if err != nil {
		if strings.Contains(err.Error(), "Process exited with status 255") == true {
			log.Printf("[resourceVIRTUALDISKDelete] Already deleted:%s", virtualDiskID)
		} else {
			// todo more descriptive err message
			log.Printf("[resourceVIRTUALDISKDelete] Failed destroy virtual disk id: %s\n", stdout)
			return err
		}
	}

	//  Delete dir if it's empty
	remoteCmd = fmt.Sprintf("ls -al \"/vmfs/volumes/%s/%s/\" |wc -l", virtualDiskDiskStore, virtualDiskDir)
	stdout, err = RunHostCommand(esxiSSHinfo, remoteCmd, "Check if Storage dir is empty")
	if stdout == "3" {
		{
			//  Delete empty dir.  Ignore stdout and errors.
			remoteCmd = fmt.Sprintf("rmdir \"/vmfs/volumes/%s/%s\"", virtualDiskDiskStore, virtualDiskDir)
			_, _ = RunHostCommand(esxiSSHinfo, remoteCmd, "rmdir empty Storage dir")
		}
	}

	d.SetId("")
	return nil
}
