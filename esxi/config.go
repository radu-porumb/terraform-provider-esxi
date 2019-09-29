package esxi

import (
	"fmt"
	"log"
)

// Config struct contains configuration data for the ESXi host
type Config struct {
	esxiHostName string
	esxiHostPort string
	esxiUserName string
	esxiPassword string
}

// validateEsxiCredentials tests the ESXi credentials by attempting to connect to ESXi host
func (c *Config) validateEsxiCredentials() error {
	esxiSSHinfo := SSHConnectionSettings{c.esxiHostName, c.esxiHostPort, c.esxiUserName, c.esxiPassword}
	log.Printf("[validateEsxiCreds]\n")

	var remoteCmd string
	var err error

	remoteCmd = fmt.Sprintf("vmware --version")
	_, err = runCommandOnHost(esxiSSHinfo, remoteCmd, "Connectivity test, get vmware version")
	if err != nil {
		return fmt.Errorf("Failed to connect to esxi host: %s", err)
	}
	return nil
}
