package esxi

import (
	"bufio"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
)

// ReadGuestDataIntoResource reads the guest VM data into the resource struct
func ReadGuestDataIntoResource(d *schema.ResourceData, m interface{}) error {
	c := m.(*Config)
	log.Println("[resourceGUESTRead]")

	guestStartupTimeout := d.Get("guest_startup_timeout").(int)

	guestName, diskStore, diskSize, bootDiskType, resourcePoolName, memsize, numvcpus, virthwver, guestos, ipAddress, virtualNetworks, virtualDisks, power, notes, guestinfo, err := ReadGuestVMData(c, d.Id(), guestStartupTimeout)

	if err != nil || guestName == "" {
		d.SetId("")
		return nil
	}

	d.Set("guest_name", guestName)
	d.Set("disk_store", diskStore)
	d.Set("disk_size", diskSize)
	if bootDiskType != "Unknown" && bootDiskType != "" {
		d.Set("boot_disk_type", bootDiskType)
	}
	d.Set("resource_pool_name", resourcePoolName)
	d.Set("memsize", memsize)
	d.Set("numvcpus", numvcpus)
	d.Set("virthwver", virthwver)
	d.Set("guestos", guestos)
	d.Set("ip_address", ipAddress)
	d.Set("power", power)
	d.Set("notes", notes)
	d.Set("guestinfo", guestinfo)

	if d.Get("guest_startup_timeout").(int) > 1 {
		d.Set("guest_startup_timeout", d.Get("guest_startup_timeout").(int))
	} else {
		d.Set("guest_startup_timeout", 60)
	}
	if d.Get("guest_shutdown_timeout").(int) > 0 {
		d.Set("guest_shutdown_timeout", d.Get("guest_shutdown_timeout").(int))
	} else {
		d.Set("guest_shutdown_timeout", 20)
	}

	// Do network interfaces
	log.Printf("virtual_networks: %q\n", virtualNetworks)
	nics := make([]map[string]interface{}, 0, 1)

	if virtualNetworks[0][0] == "" {
		nics = nil
	}

	for nic := 0; nic < 10; nic++ {
		if virtualNetworks[nic][0] != "" {
			out := make(map[string]interface{})
			out["virtual_network"] = virtualNetworks[nic][0]
			out["mac_address"] = virtualNetworks[nic][1]
			out["nic_type"] = virtualNetworks[nic][2]
			nics = append(nics, out)
		}
	}
	d.Set("network_interfaces", nics)

	// Do virtual disks
	log.Printf("virtual_disks: %q\n", virtualDisks)
	vdisks := make([]map[string]interface{}, 0, 1)

	if virtualDisks[0][0] == "" {
		vdisks = nil
	}

	for vdisk := 0; vdisk < 60; vdisk++ {
		if virtualDisks[vdisk][0] != "" {
			out := make(map[string]interface{})
			out["virtual_disk_id"] = virtualDisks[vdisk][0]
			out["slot"] = virtualDisks[vdisk][1]
			vdisks = append(vdisks, out)
		}
	}
	d.Set("virtual_disks", vdisks)

	return nil
}

// ReadGuestVMData reads the data of a guest VM from the host
func ReadGuestVMData(c *Config, vmid string, guestStartupTimeout int) (string, string, string, string, string, string, string, string, string, string, [10][3]string, [60][2]string, string, string, map[string]interface{}, error) {
	esxiSSHinfo := SSHConnectionSettings{c.esxiHostName, c.esxiHostPort, c.esxiUserName, c.esxiPassword}
	log.Println("[guestREAD]")

	var guestName, diskStore, virtualDiskType, resourcePoolName, guestos, ipAddress, notes string
	var destVmxDiskStore, destVmx, destVmxAbsolutePath, vmxContent, power string
	var diskSize, vdiskindex int
	var memsize, numvcpus, virthwver string
	var virtualNetworks [10][3]string
	var virtualDisks [60][2]string
	var guestinfo map[string]interface{}

	r, _ := regexp.Compile("")

	remoteCmd := fmt.Sprintf("vim-cmd  vmsvc/get.summary %s", vmid)
	stdout, err := RunHostCommand(esxiSSHinfo, remoteCmd, "Get Guest summary")

	if strings.Contains(stdout, "Unable to find a VM corresponding") {
		return "", "", "", "", "", "", "", "", "", "", virtualNetworks, virtualDisks, "", "", nil, nil
	}

	scanner := bufio.NewScanner(strings.NewReader(stdout))
	for scanner.Scan() {
		switch {
		case strings.Contains(scanner.Text(), "name = "):
			r, _ = regexp.Compile(`\".*\"`)
			guestName = r.FindString(scanner.Text())
			nr := strings.NewReplacer(`"`, "", `"`, "")
			guestName = nr.Replace(guestName)
		case strings.Contains(scanner.Text(), "vmPathName = "):
			r, _ = regexp.Compile(`\[.*\]`)
			diskStore = r.FindString(scanner.Text())
			nr := strings.NewReplacer("[", "", "]", "")
			diskStore = nr.Replace(diskStore)
		}
	}

	//  Get resource pool that this VM is located
	remoteCmd = fmt.Sprintf(`grep -A2 'objID>%s</objID' /etc/vmware/hostd/pools.xml | grep -o resourcePool.*resourcePool`, vmid)
	stdout, err = RunHostCommand(esxiSSHinfo, remoteCmd, "check if guest is in resource pool")
	nr := strings.NewReplacer("resourcePool>", "", "</resourcePool", "")
	vmResourcePoolID := nr.Replace(stdout)
	log.Printf("[GuestRead] resource_pool_name|%s| scanner.Text():|%s|\n", vmResourcePoolID, stdout)
	resourcePoolName, err = getResourcePoolName(c, vmResourcePoolID)
	log.Printf("[GuestRead] resource_pool_name|%s| scanner.Text():|%s|\n", vmResourcePoolID, err)

	//
	//  Read vmx file into memory to read settings
	//
	//      -Get location of vmx file on esxi host
	remoteCmd = fmt.Sprintf("vim-cmd vmsvc/get.config %s | grep vmPathName|grep -oE \"\\[.*\\]\"", vmid)
	stdout, err = RunHostCommand(esxiSSHinfo, remoteCmd, "get dst_vmx_ds")
	destVmxDiskStore = stdout
	destVmxDiskStore = strings.Trim(destVmxDiskStore, "[")
	destVmxDiskStore = strings.Trim(destVmxDiskStore, "]")

	remoteCmd = fmt.Sprintf("vim-cmd vmsvc/get.config %s | grep vmPathName|awk '{print $NF}'|sed 's/[\"|,]//g'", vmid)
	stdout, err = RunHostCommand(esxiSSHinfo, remoteCmd, "get dst_vmx")
	destVmx = stdout

	destVmxAbsolutePath = "/vmfs/volumes/" + destVmxDiskStore + "/" + destVmx

	log.Printf("[guestREAD] dst_vmx_file: %s\n", destVmxAbsolutePath)
	log.Printf("[guestREAD] disk_store: %s  dst_vmx_ds:%s\n", diskStore, destVmxAbsolutePath)

	remoteCmd = fmt.Sprintf("cat \"%s\"", destVmxAbsolutePath)
	vmxContent, err = RunHostCommand(esxiSSHinfo, remoteCmd, "read guest_name.vmx file")

	// Used to keep track if a network interface is using static or generated macs.
	var isGeneratedMAC [10]bool

	//  Read vmx_contents line-by-line to get current settings.
	vdiskindex = 0
	scanner = bufio.NewScanner(strings.NewReader(vmxContent))
	for scanner.Scan() {

		switch {
		case strings.Contains(scanner.Text(), "memSize = "):
			r, _ = regexp.Compile(`\".*\"`)
			stdout = r.FindString(scanner.Text())
			nr = strings.NewReplacer(`"`, "", `"`, "")
			memsize = nr.Replace(stdout)
			log.Printf("[guestREAD] memsize found: %s\n", memsize)

		case strings.Contains(scanner.Text(), "numvcpus = "):
			r, _ = regexp.Compile(`\".*\"`)
			stdout = r.FindString(scanner.Text())
			nr = strings.NewReplacer(`"`, "", `"`, "")
			numvcpus = nr.Replace(stdout)
			log.Printf("[guestREAD] numvcpus found: %s\n", numvcpus)

		case strings.Contains(scanner.Text(), "numa.autosize.vcpu."):
			r, _ = regexp.Compile(`\".*\"`)
			stdout = r.FindString(scanner.Text())
			nr = strings.NewReplacer(`"`, "", `"`, "")
			numvcpus = nr.Replace(stdout)
			log.Printf("[guestREAD] numa.vcpu (numvcpus) found: %s\n", numvcpus)

		case strings.Contains(scanner.Text(), "virtualHW.version = "):
			r, _ = regexp.Compile(`\".*\"`)
			stdout = r.FindString(scanner.Text())
			virthwver = strings.Replace(stdout, `"`, "", -1)
			log.Printf("[guestREAD] virthwver found: %s\n", virthwver)

		case strings.Contains(scanner.Text(), "guestOS = "):
			r, _ = regexp.Compile(`\".*\"`)
			stdout = r.FindString(scanner.Text())
			guestos = strings.Replace(stdout, `"`, "", -1)
			log.Printf("[guestREAD] guestos found: %s\n", guestos)

		case strings.Contains(scanner.Text(), "scsi"):
			re := regexp.MustCompile("scsi([0-3]):([0-9]{1,2}).(.*) = \"(.*)\"")
			results := re.FindStringSubmatch(scanner.Text())
			if len(results) > 4 {
				log.Printf("[guestREAD] %s : %s . %s = %s\n", results[1], results[2], results[3], results[4])

				if (results[1] == "0") && (results[2] == "0") {
					// Skip boot disk
				} else {
					if strings.Contains(results[3], "fileName") == true {
						log.Printf("[guestREAD] %s : %s\n", results[0], results[4])
						virtualDisks[vdiskindex][0] = results[4]
						virtualDisks[vdiskindex][1] = fmt.Sprintf("%s:%s", results[1], results[2])
						vdiskindex++
					}
				}
			}

		case strings.Contains(scanner.Text(), "ethernet"):
			re := regexp.MustCompile("ethernet(.).(.*) = \"(.*)\"")
			results := re.FindStringSubmatch(scanner.Text())
			index, _ := strconv.Atoi(results[1])

			switch results[2] {
			case "networkName":
				virtualNetworks[index][0] = results[3]
				log.Printf("[guestREAD] %s : %s\n", results[0], results[3])

			case "addressType":
				if results[3] == "generated" {
					isGeneratedMAC[index] = true
				}

			case "generatedAddress":
				if isGeneratedMAC[index] == true {
					virtualNetworks[index][1] = results[3]
					log.Printf("[guestREAD] %s : %s\n", results[0], results[3])
				}

			case "address":
				if isGeneratedMAC[index] == false {
					virtualNetworks[index][1] = results[3]
					log.Printf("[resourceGUESTRead] %s : %s\n", results[0], results[3])
				}

			case "virtualDev":
				virtualNetworks[index][2] = results[3]
				log.Printf("[guestREAD] %s : %s\n", results[0], results[3])
			}

		case strings.Contains(scanner.Text(), "annotation = "):
			r, _ = regexp.Compile(`\".*\"`)
			stdout = r.FindString(scanner.Text())
			notes = strings.Replace(stdout, `"`, "", -1)
			notes = strings.Replace(notes, "|22", "\"", -1)
			log.Printf("[guestREAD] annotation found: %s\n", notes)

		}
	}

	parsedVmx := ParseVmxFile(vmxContent)

	//  Get power state
	log.Println("guestREAD: guestPowerGetState")
	power = GetGuestPowerState(c, vmid)

	//
	// Get IP address (need vmware tools installed)
	//
	if power == "on" {
		ipAddress = GetGuestIPAddress(c, vmid, guestStartupTimeout)
		log.Printf("[guestREAD] guestGetIpAddress: %s\n", ipAddress)
	} else {
		ipAddress = ""
	}

	// Get boot disk size
	bootDiskPath, _ := GetBootDiskPath(c, vmid)
	_, _, _, diskSize, virtualDiskType, err = ReadVirtualDiskInfo(c, bootDiskPath)
	diskSizeString := strconv.Itoa(diskSize)

	// Get guestinfo value
	guestinfo = make(map[string]interface{})
	for key, value := range parsedVmx {
		if strings.Contains(key, "guestinfo") {
			shortKey := strings.Replace(key, "guestinfo.", "", -1)
			guestinfo[shortKey] = value
		}
	}

	// return results
	return guestName, diskStore, diskSizeString, virtualDiskType, resourcePoolName, memsize, numvcpus, virthwver, guestos, ipAddress, virtualNetworks, virtualDisks, power, notes, guestinfo, err
}
