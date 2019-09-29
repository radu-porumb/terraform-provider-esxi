package esxi

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// getGuestVMID gets the guest VM's ID by the name
func getGuestVMID(c *Config, guestName string) (string, error) {
	esxiSSHinfo := SSHConnectionSettings{c.esxiHostName, c.esxiHostPort, c.esxiUserName, c.esxiPassword}
	log.Printf("[guestGetVMID]\n")

	var remoteCmd, vmid string
	var err error

	remoteCmd = fmt.Sprintf("vim-cmd vmsvc/getallvms 2>/dev/null | sort -n | "+
		"grep \"[0-9] * %s .*%s\" | awk '{print $1}' | "+
		"tail -1", guestName, guestName)

	vmid, err = runCommandOnHost(esxiSSHinfo, remoteCmd, "get vmid")
	log.Printf("[guestGetVMID] result: %s\n", vmid)
	if err != nil {
		log.Printf("[guestGetVMID] Failed get vmid: %s\n", err)
		return "", fmt.Errorf("Failed get vmid: %s", err)
	}

	return vmid, nil
}

// validateGuestVMID validates a guest VM's ID
func validateGuestVMID(c *Config, vmid string) (string, error) {
	esxiSSHinfo := SSHConnectionSettings{c.esxiHostName, c.esxiHostPort, c.esxiUserName, c.esxiPassword}
	log.Printf("[guestValidateVMID]\n")

	var remoteCmd string
	var err error

	remoteCmd = fmt.Sprintf("vim-cmd vmsvc/getallvms 2>/dev/null | awk '{print $1}' | "+
		"grep '^%s$'", vmid)

	vmid, err = runCommandOnHost(esxiSSHinfo, remoteCmd, "validate vmid exists")
	log.Printf("[guestValidateVMID] result: %s\n", vmid)
	if err != nil {
		log.Printf("[guestValidateVMID] Failed get vmid: %s\n", err)
		return "", fmt.Errorf("Failed get vmid: %s", err)
	}

	return vmid, nil
}

// getBootDiskPath gets the path of the VM's book disk VMDK
func getBootDiskPath(c *Config, vmid string) (string, error) {
	esxiSSHinfo := SSHConnectionSettings{c.esxiHostName, c.esxiHostPort, c.esxiUserName, c.esxiPassword}
	log.Printf("[getBootDiskPath]\n")

	var remoteCmd, stdout string
	var err error

	remoteCmd = fmt.Sprintf("vim-cmd vmsvc/device.getdevices %s | grep -A10 'key = 2000'|grep -m 1 fileName", vmid)
	stdout, err = runCommandOnHost(esxiSSHinfo, remoteCmd, "get boot disk")
	if err != nil {
		log.Printf("[getBootDiskPath] Failed get boot disk path: %s\n", stdout)
		return "Failed get boot disk path:", err
	}
	r := strings.NewReplacer("fileName = \"[", "/vmfs/volumes/",
		"] ", "/", "\",", "")
	return r.Replace(stdout), err
}

// getDestVmxAbsPath gets the absolute path for the VMX file on the host
func getDestVmxAbsPath(c *Config, vmid string) (string, error) {
	esxiSSHinfo := SSHConnectionSettings{c.esxiHostName, c.esxiHostPort, c.esxiUserName, c.esxiPassword}
	log.Printf("[getDst_vmx_file]\n")

	var destVmxDiskStore, destVmxPath, destVmxAbsPath string

	//      -Get location of vmx file on esxi host
	remoteCmd := fmt.Sprintf("vim-cmd vmsvc/get.config %s | grep vmPathName|grep -oE \"\\[.*\\]\"", vmid)
	stdout, err := runCommandOnHost(esxiSSHinfo, remoteCmd, "get dst_vmx_ds")
	destVmxDiskStore = stdout
	destVmxDiskStore = strings.Trim(destVmxDiskStore, "[")
	destVmxDiskStore = strings.Trim(destVmxDiskStore, "]")

	remoteCmd = fmt.Sprintf("vim-cmd vmsvc/get.config %s | grep vmPathName|awk '{print $NF}'|sed 's/[\"|,]//g'", vmid)
	stdout, err = runCommandOnHost(esxiSSHinfo, remoteCmd, "get dst_vmx")
	destVmxPath = stdout

	destVmxAbsPath = "/vmfs/volumes/" + destVmxDiskStore + "/" + destVmxPath
	return destVmxAbsPath, err
}

// readVmxContent reads the content of a VMX file on the host machine
func readVmxContent(c *Config, vmid string) (string, error) {
	esxiSSHinfo := SSHConnectionSettings{c.esxiHostName, c.esxiHostPort, c.esxiUserName, c.esxiPassword}
	log.Printf("[getVmx_contents]\n")

	var remoteCmd, vmxContent string

	destVmxFile, err := getDestVmxAbsPath(c, vmid)
	remoteCmd = fmt.Sprintf("cat \"%s\"", destVmxFile)
	vmxContent, err = runCommandOnHost(esxiSSHinfo, remoteCmd, "read guest_name.vmx file")

	return vmxContent, err
}

// updateVmx updates the VMX file on the host
func updateVmx(c *Config, vmid string, iscreate bool, memsize int, numvcpus int,
	virthwver int, guestos string, virtualNetworks [10][3]string, virtualDisks [60][2]string, notes string,
	guestinfo map[string]interface{}) error {

	esxiSSHinfo := SSHConnectionSettings{c.esxiHostName, c.esxiHostPort, c.esxiUserName, c.esxiPassword}
	log.Printf("[updateVmx_contents]\n")

	var regexReplacement, remoteCmd string

	vmxContent, err := readVmxContent(c, vmid)
	if err != nil {
		log.Printf("[updateVmx_contents] Failed get vmx contents: %s\n", err)
		return err
	}
	if strings.Contains(vmxContent, "Unable to find a VM corresponding") {
		return nil
	}

	// modify memsize
	if memsize != 0 {
		re := regexp.MustCompile("memSize = \".*\"")
		regexReplacement = fmt.Sprintf("memSize = \"%d\"", memsize)
		vmxContent = re.ReplaceAllString(vmxContent, regexReplacement)
	}

	// modify numvcpus
	if numvcpus != 0 {
		re := regexp.MustCompile("numvcpus = \".*\"")
		regexReplacement = fmt.Sprintf("numvcpus = \"%d\"", numvcpus)
		vmxContent = re.ReplaceAllString(vmxContent, regexReplacement)
	}

	// modify virthwver
	if virthwver != 0 {
		re := regexp.MustCompile("virtualHW.version = \".*\"")
		regexReplacement = fmt.Sprintf("virtualHW.version = \"%d\"", virthwver)
		vmxContent = re.ReplaceAllString(vmxContent, regexReplacement)
	}

	// modify guestos
	if guestos != "" {
		re := regexp.MustCompile("guestOS = \".*\"")
		regexReplacement = fmt.Sprintf("guestOS = \"%s\"", guestos)
		vmxContent = re.ReplaceAllString(vmxContent, regexReplacement)
	}

	// modify annotation
	if notes != "" {
		notes = strings.Replace(notes, "\"", "|22", -1)
		if strings.Contains(vmxContent, "annotation") {
			re := regexp.MustCompile("annotation = \".*\"")
			regexReplacement = fmt.Sprintf("annotation = \"%s\"", notes)
			vmxContent = re.ReplaceAllString(vmxContent, regexReplacement)
		} else {
			regexReplacement = fmt.Sprintf("\nannotation = \"%s\"", notes)
			vmxContent += regexReplacement
		}
	}

	if len(guestinfo) > 0 {
		parsedVmx := parseVmxFile(vmxContent)
		for k, v := range guestinfo {
			log.Println("SAVING", k, v)
			parsedVmx["guestinfo."+k] = v.(string)
		}
		vmxContent = buildVmxString(parsedVmx)
	}

	//
	//  add/modify virtual disks
	//
	var tmpvar string
	var newVmxContent string
	var i, j int

	//
	//  Remove all disks
	//
	regexReplacement = fmt.Sprintf("")
	for i = 0; i < 4; i++ {
		for j = 0; j < 16; j++ {

			if (i != 0 || j != 0) && j != 7 {
				re := regexp.MustCompile(fmt.Sprintf("scsi%d:%d.*\n", i, j))
				vmxContent = re.ReplaceAllString(vmxContent, regexReplacement)
			}
		}
	}

	//
	//  Add disks that are managed by terraform
	//
	for i = 0; i < 59; i++ {
		if virtualDisks[i][0] != "" {

			log.Printf("[updateVmx_contents] Adding: %s\n", virtualDisks[i][1])
			tmpvar = fmt.Sprintf("scsi%s.deviceType = \"scsi-hardDisk\"\n", virtualDisks[i][1])
			if !strings.Contains(vmxContent, tmpvar) {
				vmxContent += "\n" + tmpvar
			}

			tmpvar = fmt.Sprintf("scsi%s.fileName", virtualDisks[i][1])
			if strings.Contains(vmxContent, tmpvar) {
				re := regexp.MustCompile(tmpvar + " = \".*\"")
				regexReplacement = fmt.Sprintf(tmpvar+" = \"%s\"", virtualDisks[i][0])
				vmxContent = re.ReplaceAllString(vmxContent, regexReplacement)
			} else {
				regexReplacement = fmt.Sprintf("\n"+tmpvar+" = \"%s\"", virtualDisks[i][0])
				vmxContent += "\n" + regexReplacement
			}

			tmpvar = fmt.Sprintf("scsi%s.present = \"true\"\n", virtualDisks[i][1])
			if !strings.Contains(vmxContent, tmpvar) {
				vmxContent += "\n" + tmpvar
			}

		}
	}

	//
	//  Create/update networks network_interfaces
	//

	//  Define default nic type.
	var defaultNetworkType, networkType string
	if virtualNetworks[0][2] != "" {
		defaultNetworkType = virtualNetworks[0][2]
	} else {
		defaultNetworkType = "e1000"
	}

	//  If this is first time provisioning, delete all the old ethernet configuration.
	if iscreate == true {
		log.Printf("[updateVmx_contents] Delete old ethernet configuration\n")
		regexReplacement = fmt.Sprintf("")
		for i = 0; i < 9; i++ {
			re := regexp.MustCompile(fmt.Sprintf("ethernet%d.*\n", i))
			vmxContent = re.ReplaceAllString(vmxContent, regexReplacement)
		}
	}

	//  Add/Modify virtual networks.
	networkType = ""

	for i := 0; i <= 9; i++ {
		log.Printf("[updateVmx_contents] ethernet%d\n", i)

		if virtualNetworks[i][0] == "" && strings.Contains(vmxContent, "ethernet"+strconv.Itoa(i)) == true {
			//  This is Modify (Delete existing network configuration)
			log.Printf("[updateVmx_contents] ethernet%d Delete existing.\n", i)
			regexReplacement = fmt.Sprintf("")
			re := regexp.MustCompile(fmt.Sprintf("ethernet%d.*\n", i))
			vmxContent = re.ReplaceAllString(vmxContent, regexReplacement)
		}

		if virtualNetworks[i][0] != "" && strings.Contains(vmxContent, "ethernet"+strconv.Itoa(i)) == true {
			//  This is Modify
			log.Printf("[updateVmx_contents] ethernet%d Modify existing.\n", i)

			//  Modify Network Name
			re := regexp.MustCompile("ethernet" + strconv.Itoa(i) + ".networkName = \".*\"")
			regexReplacement = fmt.Sprintf("ethernet"+strconv.Itoa(i)+".networkName = \"%s\"", virtualNetworks[i][0])
			vmxContent = re.ReplaceAllString(vmxContent, regexReplacement)

			//  Modify virtual Device
			re = regexp.MustCompile("ethernet" + strconv.Itoa(i) + ".virtualDev = \".*\"")
			regexReplacement = fmt.Sprintf("ethernet"+strconv.Itoa(i)+".virtualDev = \"%s\"", virtualNetworks[i][2])
			vmxContent = re.ReplaceAllString(vmxContent, regexReplacement)

			//  Modify MAC  todo
		}

		if virtualNetworks[i][0] != "" && strings.Contains(vmxContent, "ethernet"+strconv.Itoa(i)) == false {
			//  This is create

			//  Set virtual_network name
			log.Printf("[updateVmx_contents] ethernet%d Create New: %s\n", i, virtualNetworks[i][0])
			tmpvar = fmt.Sprintf("\nethernet%d.networkName = \"%s\"\n", i, virtualNetworks[i][0])
			newVmxContent = tmpvar

			//  Set mac address
			if virtualNetworks[i][1] != "" {
				tmpvar = fmt.Sprintf("ethernet%d.addressType = \"static\"\n", i)
				newVmxContent = newVmxContent + tmpvar

				tmpvar = fmt.Sprintf("ethernet%d.address = \"%s\"\n", i, virtualNetworks[i][1])
				newVmxContent = newVmxContent + tmpvar
			}

			//  Set network type
			if virtualNetworks[i][2] == "" {
				networkType = defaultNetworkType
			} else {
				networkType = virtualNetworks[i][2]
			}

			tmpvar = fmt.Sprintf("ethernet%d.virtualDev = \"%s\"\n", i, networkType)
			newVmxContent = newVmxContent + tmpvar

			tmpvar = fmt.Sprintf("ethernet%d.present = \"TRUE\"\n", i)

			vmxContent = vmxContent + newVmxContent + tmpvar
		}
	}

	//
	//  Write vmx file to esxi host
	//
	log.Printf("[updateVmx_contents] New guest_name.vmx: %s\n", vmxContent)

	destVmxFilePath, err := getDestVmxAbsPath(c, vmid)
	vmxFileName := getVmxFileFromPath(destVmxFilePath)

	if err != nil {
		log.Printf("[updateVmx_contents] Failed to get VMX file name from ESXi: %s\n", err)
		return err
	}

	err = saveVmxStringToDisk(vmxFileName, vmxContent)

	if err != nil {
		return err
	}

	err = copyFileToHost(esxiSSHinfo, vmxFileName, destVmxFilePath)

	if err != nil {
		return err
	}

	err = deleteVmx(vmxFileName)

	if err != nil {
		return err
	}

	remoteCmd = fmt.Sprintf("vim-cmd vmsvc/reload %s", vmid)
	_, err = runCommandOnHost(esxiSSHinfo, remoteCmd, "vmsvc/reload")
	return err
}

// cleanVmxStorage cleans the VMX file storage data
func cleanVmxStorage(c *Config, vmid string) error {
	esxiSSHinfo := SSHConnectionSettings{c.esxiHostName, c.esxiHostPort, c.esxiUserName, c.esxiPassword}
	log.Printf("[cleanStorageFromVmx]\n")

	var remoteCmd string

	vmxContent, err := readVmxContent(c, vmid)
	if err != nil {
		log.Printf("[updateVmx_contents] Failed get vmx contents: %s\n", err)
		return err
	}

	for x := 0; x < 4; x++ {
		for y := 0; y < 16; y++ {
			if !(x == 0 && y == 0) {
				regexReplacement := fmt.Sprintf("scsi%d:%d.*", x, y)
				re := regexp.MustCompile(regexReplacement)
				vmxContent = re.ReplaceAllString(vmxContent, "")
			}
		}
	}

	//
	//  Write vmx file to esxi host
	//
	vmxContent = strings.Replace(vmxContent, "\"", "\\\"", -1)

	destVmxFile, err := getDestVmxAbsPath(c, vmid)

	remoteCmd = fmt.Sprintf("echo \"%s\" | grep '[^[:blank:]]' >%s", vmxContent, destVmxFile)
	vmxContent, err = runCommandOnHost(esxiSSHinfo, remoteCmd, "write guest_name.vmx file")

	remoteCmd = fmt.Sprintf("vim-cmd vmsvc/reload %s", vmid)
	_, err = runCommandOnHost(esxiSSHinfo, remoteCmd, "vmsvc/reload")
	return err
}

// powerOnGuest powers on the guest VM
func powerOnGuest(c *Config, vmid string) (string, error) {
	esxiSSHinfo := SSHConnectionSettings{c.esxiHostName, c.esxiHostPort, c.esxiUserName, c.esxiPassword}
	log.Printf("[guestPowerOn]\n")

	if getGuestPowerState(c, vmid) == "on" {
		return "", nil
	}

	remoteCmd := fmt.Sprintf("vim-cmd vmsvc/power.on %s", vmid)
	stdout, err := runCommandOnHost(esxiSSHinfo, remoteCmd, "vmsvc/power.on")
	time.Sleep(3 * time.Second)

	if getGuestPowerState(c, vmid) == "on" {
		return stdout, nil
	}

	return stdout, err
}

// powerOffGuest powers off the guest VM
func powerOffGuest(c *Config, vmid string, guestShutdownTimeout int) (string, error) {
	esxiSSHinfo := SSHConnectionSettings{c.esxiHostName, c.esxiHostPort, c.esxiUserName, c.esxiPassword}
	log.Printf("[guestPowerOff]\n")

	var remoteCmd, stdout string

	savedpowerstate := getGuestPowerState(c, vmid)
	if savedpowerstate == "off" {
		return "", nil

	} else if savedpowerstate == "on" {

		if guestShutdownTimeout != 0 {
			remoteCmd = fmt.Sprintf("vim-cmd vmsvc/power.shutdown %s", vmid)
			stdout, _ = runCommandOnHost(esxiSSHinfo, remoteCmd, "vmsvc/power.shutdown")
			time.Sleep(3 * time.Second)

			for i := 0; i < (guestShutdownTimeout / 3); i++ {
				if getGuestPowerState(c, vmid) == "off" {
					return stdout, nil
				}
				time.Sleep(3 * time.Second)
			}
		}

		remoteCmd = fmt.Sprintf("vim-cmd vmsvc/power.off %s", vmid)
		stdout, _ = runCommandOnHost(esxiSSHinfo, remoteCmd, "vmsvc/power.off")
		time.Sleep(1 * time.Second)

		return stdout, nil

	} else {
		remoteCmd = fmt.Sprintf("vim-cmd vmsvc/power.off %s", vmid)
		stdout, _ = runCommandOnHost(esxiSSHinfo, remoteCmd, "vmsvc/power.off")
		return stdout, nil
	}
}

// getGuestPowerState returns whether the guest VM is powered on or off
func getGuestPowerState(c *Config, vmid string) string {
	esxiSSHinfo := SSHConnectionSettings{c.esxiHostName, c.esxiHostPort, c.esxiUserName, c.esxiPassword}
	log.Printf("[guestPowerGetState]\n")

	remoteCmd := fmt.Sprintf("vim-cmd vmsvc/power.getstate %s", vmid)
	stdout, _ := runCommandOnHost(esxiSSHinfo, remoteCmd, "vmsvc/power.getstate")
	if strings.Contains(stdout, "Unable to find a VM corresponding") {
		return "Unknown"
	}

	if strings.Contains(stdout, "Powered off") == true {
		return "off"
	} else if strings.Contains(stdout, "Powered on") == true {
		return "on"
	} else if strings.Contains(stdout, "Suspended") == true {
		return "suspended"
	} else {
		return "Unknown"
	}
}

// getGuestIPAddress gets the guest VM's IP address
func getGuestIPAddress(c *Config, vmid string, guestStartupTimeout int) string {
	esxiSSHinfo := SSHConnectionSettings{c.esxiHostName, c.esxiHostPort, c.esxiUserName, c.esxiPassword}
	log.Printf("[guestGetIpAddress]\n")

	var remoteCmd, stdout, ipAddress, ipAddress2 string
	var uptime int

	//  Check if powered off
	if getGuestPowerState(c, vmid) != "on" {
		return ""
	}

	//
	//  Check uptime of guest.
	//
	uptime = 0
	for uptime < guestStartupTimeout {
		//  Primary method to get IP
		remoteCmd = fmt.Sprintf("vim-cmd vmsvc/get.guest %s 2>/dev/null |grep -A 5 'deviceConfigId = 4000' |tail -1|grep -oE '((1?[0-9][0-9]?|2[0-4][0-9]|25[0-5]).){3}(1?[0-9][0-9]?|2[0-4][0-9]|25[0-5])'", vmid)
		stdout, _ = runCommandOnHost(esxiSSHinfo, remoteCmd, "get ip_address method 1")
		ipAddress = stdout
		if ipAddress != "" {
			return ipAddress
		}

		time.Sleep(3 * time.Second)

		//  Get uptime if above failed.
		remoteCmd = fmt.Sprintf("vim-cmd vmsvc/get.summary %s 2>/dev/null | grep 'uptimeSeconds ='|sed 's/^.*= //g'|sed s/,//g", vmid)
		stdout, err := runCommandOnHost(esxiSSHinfo, remoteCmd, "get uptime")
		if err != nil {
			return ""
		}
		uptime, _ = strconv.Atoi(stdout)
	}

	//
	// Alternate method to get IP
	//
	remoteCmd = fmt.Sprintf("vim-cmd vmsvc/get.summary %s 2>/dev/null | grep 'uptimeSeconds ='|sed 's/^.*= //g'|sed s/,//g", vmid)
	stdout, _ = runCommandOnHost(esxiSSHinfo, remoteCmd, "get uptime")
	uptime, _ = strconv.Atoi(stdout)
	if uptime > 120 {
		remoteCmd = fmt.Sprintf("vim-cmd vmsvc/get.guest %s 2>/dev/null | grep -m 1 '^   ipAddress = ' | grep -oE '((1?[0-9][0-9]?|2[0-4][0-9]|25[0-5]).){3}(1?[0-9][0-9]?|2[0-4][0-9]|25[0-5])'", vmid)
		stdout, _ = runCommandOnHost(esxiSSHinfo, remoteCmd, "get ip_address method 2")
		ipAddress2 = stdout
		if ipAddress2 != "" {
			return ipAddress2
		}
	}

	return ""
}
