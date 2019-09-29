package esxi

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
)

func deleteResourcePoolResource(d *schema.ResourceData, m interface{}) error {
	c := m.(*Config)
	esxiSSHinfo := SSHConnectionSettings{c.esxiHostName, c.esxiHostPort, c.esxiUserName, c.esxiPassword}
	log.Println("[resourceRESOURCEPOOLDelete]")

	var remoteCmd, stdout string
	var err error

	poolID := d.Id()

	remoteCmd = fmt.Sprintf("vim-cmd hostsvc/rsrc/destroy %s", poolID)
	stdout, err = runCommandOnHost(esxiSSHinfo, remoteCmd, "destroy resource pool")
	if err != nil {
		// todo more descriptive err message
		log.Printf("[resourcePoolDELETE] Failed destroy resource pool id: %s\n", stdout)
		return err
	}

	d.SetId("")
	return nil
}
