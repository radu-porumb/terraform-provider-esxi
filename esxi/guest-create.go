package esxi

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
)

// createGuestResource creates the guest resource
func createGuestResource(d *schema.ResourceData, m interface{}) error {
	c := m.(*Config)

	log.Printf("[resourceGUESTCreate]\n")

	var virtualNetworks [10][3]string
	var virtualDisks [60][2]string
	var srcPath string
	var tmpint, i, virtualDiskCount int

	cloneFromVM := d.Get("clone_from_vm").(string)
	ovfSource := d.Get("ovf_source").(string)
	diskStore := d.Get("disk_store").(string)
	resourcePoolName := d.Get("resource_pool_name").(string)
	guestName := d.Get("guest_name").(string)
	bootDiskType := d.Get("boot_disk_type").(string)
	bootDiskSize := d.Get("boot_disk_size").(string)
	memsize := d.Get("memsize").(string)
	numvcpus := d.Get("numvcpus").(string)
	virthwver := d.Get("virthwver").(string)
	guestos := d.Get("guestos").(string)
	notes := d.Get("notes").(string)
	power := d.Get("power").(string)
	guestShutdownTimeout := d.Get("guest_shutdown_timeout").(int)

	guestinfo, ok := d.Get("guestinfo").(map[string]interface{})
	if !ok {
		return errors.New("guestinfo is wrong type")
	}

	// Validations
	if resourcePoolName == "ha-root-pool" {
		resourcePoolName = "/"
	}

	if cloneFromVM != "" {
		password := url.QueryEscape(c.esxiPassword)
		srcPath = fmt.Sprintf("vi://%s:%s@%s/%s", c.esxiUserName, password, c.esxiHostName, cloneFromVM)
	} else if ovfSource != "" {
		srcPath = ovfSource
	} else {
		srcPath = "none"
	}

	//  Validate number of virthwver.
	// todo
	//switch virthwver {
	//case 0,4,7,8,9,10,11,12,13,14:
	//  // virthwver check passes.
	//default:
	//  return errors.New("Error: virthwver must be 4,7,8,9,10,11,12,13 or 14")
	//}

	//  Validate guestos
	if validateGuestOsType(guestos) == false {
		return errors.New("Error: invalid guestos.  see https://github.com/josenk/vagrant-vmware-esxi/wiki/VMware-ESXi-6.5-guestOS-types")
	}

	// Validate boot_disk_type
	if bootDiskType == "" {
		bootDiskType = "thin"
	}
	if bootDiskType != "thin" && bootDiskType != "zeroedthick" && bootDiskType != "eagerzeroedthick" {
		return errors.New("Error: boot_disk_type must be thin, zeroedthick or eagerzeroedthick")
	}

	//  Validate boot_disk_size.
	if _, err := strconv.Atoi(bootDiskSize); err != nil && bootDiskSize != "" {
		return errors.New("Error: boot_disk_size must be an integer")
	}
	tmpint, _ = strconv.Atoi(bootDiskSize)
	if (tmpint < 1 || tmpint > 62000) && bootDiskSize != "" {
		return errors.New("Error: boot_disk_size must be an > 1 and < 62000")
	}

	//  Validate lan adapters
	lanAdaptersCount := d.Get("network_interfaces.#").(int)
	if lanAdaptersCount > 10 {
		lanAdaptersCount = 10
	}
	for i = 0; i < lanAdaptersCount; i++ {
		prefix := fmt.Sprintf("network_interfaces.%d.", i)

		if attr, ok := d.Get(prefix + "virtual_network").(string); ok && attr != "" {
			virtualNetworks[i][0] = d.Get(prefix + "virtual_network").(string)
		}

		if attr, ok := d.Get(prefix + "mac_address").(string); ok && attr != "" {
			virtualNetworks[i][1] = d.Get(prefix + "mac_address").(string)
		}

		if attr, ok := d.Get(prefix + "nic_type").(string); ok && attr != "" {
			virtualNetworks[i][2] = d.Get(prefix + "nic_type").(string)
			//  Validate nictype
			if validateNICType(virtualNetworks[i][2]) == false {
				errMSG := fmt.Sprintf("Error: invalid nic_type. %s\nMust be vlance flexible e1000 e1000e vmxnet vmxnet2 or vmxnet3", virtualNetworks[i][2])
				return errors.New(errMSG)
			}
		}
	}

	//  Validate virtual_disks
	virtualDiskCount, ok = d.Get("virtual_disks.#").(int)
	if !ok {
		virtualDiskCount = 0
		virtualDisks[0][0] = ""
	}

	if virtualDiskCount > 59 {
		virtualDiskCount = 59
	}
	for i = 0; i < virtualDiskCount; i++ {
		prefix := fmt.Sprintf("virtual_disks.%d.", i)

		if attr, ok := d.Get(prefix + "virtual_disk_id").(string); ok && attr != "" {
			virtualDisks[i][0] = d.Get(prefix + "virtual_disk_id").(string)
		}

		if attr, ok := d.Get(prefix + "slot").(string); ok && attr != "" {
			virtualDisks[i][1] = d.Get(prefix + "slot").(string)
			validateVirtualDiskSlot(virtualDisks[i][1])
			result := validateVirtualDiskSlot(virtualDisks[i][1])
			if result != "ok" {
				return errors.New(result)
			}
		}
	}

	vmid, err := createGuest(c, guestName, diskStore, srcPath, resourcePoolName, memsize,
		numvcpus, virthwver, guestos, bootDiskType, bootDiskSize, virtualNetworks,
		virtualDisks, guestShutdownTimeout, notes, guestinfo)
	if err != nil {
		tmpint, _ = strconv.Atoi(vmid)
		if tmpint > 0 {
			d.SetId(vmid)
		}
		return err
	}

	//  set vmid
	d.SetId(vmid)

	if power == "on" {
		_, err = powerOnGuest(c, vmid)
		if err != nil {
			return errors.New("Failed to power on")
		}
	}
	d.Set("power", "on")

	// Refresh
	return readGuestDataIntoResource(d, m)
}

// createGuest creates a guest VM on the host
func createGuest(c *Config, guestName string, diskStore string,
	srcPath string, resourcePoolName string, memSize string, numVCPUs string, virtHWver string, guestos string,
	bootDiskType string, bootDiskSize string, virtualNetworks [10][3]string,
	virtualDisks [60][2]string, guestShutdownTimeout int, notes string,
	guestinfo map[string]interface{}) (string, error) {

	esxiSSHinfo := SSHConnectionSettings{c.esxiHostName, c.esxiHostPort, c.esxiUserName, c.esxiPassword}
	log.Printf("[guestCREATE]\n")

	var memsize, numvcpus, virthwver int
	var bootDiskVmdkPath, remoteCmd, vmid, stdout, vmxContent string
	var osShellCmd, osShellCmdOpt string
	var out bytes.Buffer
	var err error
	err = nil

	memsize, _ = strconv.Atoi(memSize)
	numvcpus, _ = strconv.Atoi(numVCPUs)
	virthwver, _ = strconv.Atoi(virtHWver)

	//
	//  Check if Disk Store already exists
	//
	err = validateDiskStore(c, diskStore)
	if err != nil {
		return "", err
	}

	//
	//  Check if guest already exists
	//
	// get VMID (by name)
	vmid, err = getGuestVMID(c, guestName)

	if vmid != "" {
		// We don't need to create the VM.   It already exists.
		fmt.Printf("[guestCREATE] guest %s already exists vmid: %s\n", guestName, stdout)

		//
		//   Power off guest if it's powered on.
		//
		currentpowerstate := getGuestPowerState(c, vmid)
		if currentpowerstate == "on" || currentpowerstate == "suspended" {
			_, err = powerOffGuest(c, vmid, guestShutdownTimeout)
			if err != nil {
				return "", fmt.Errorf("Failed to power off existing guest. vmid:%s", vmid)
			}
		}

	} else if srcPath == "none" {

		// check if path already exists.
		fullPATH := fmt.Sprintf("\"/vmfs/volumes/%s/%s\"", diskStore, guestName)
		bootDiskVmdkPath = fmt.Sprintf("\"/vmfs/volumes/%s/%s/%s.vmdk\"", diskStore, guestName, guestName)
		remoteCmd = fmt.Sprintf("ls -d %s", fullPATH)
		stdout, _ = runCommandOnHost(esxiSSHinfo, remoteCmd, "check if guest path already exists.")
		if stdout == fullPATH {
			fmt.Printf("Error: Guest path already exists. fullPATH:%s\n", fullPATH)
			return "", fmt.Errorf("Guest path already exists. fullPATH:%s", fullPATH)
		}

		remoteCmd = fmt.Sprintf("mkdir %s", fullPATH)
		stdout, err = runCommandOnHost(esxiSSHinfo, remoteCmd, "create guest path")
		if err != nil {
			log.Printf("Failed to create guest path. fullPATH:%s\n", fullPATH)
			return "", fmt.Errorf("Failed to create guest path. fullPATH:%s", fullPATH)
		}

		hasISO := false
		isofilename := ""
		notes = strings.Replace(notes, "\"", "|22", -1)

		if numvcpus == 0 {
			numvcpus = 1
		}
		if memsize == 0 {
			memsize = 512
		}
		if virthwver == 0 {
			virthwver = 8
		}
		if guestos == "" {
			guestos = "centos-64"
		}
		if bootDiskSize == "" {
			bootDiskSize = "16"
		}

		// Build VM by default/black config
		vmxContent =
			fmt.Sprintf("config.version = \\\"8\\\"\n") +
				fmt.Sprintf("virtualHW.version = \\\"%d\\\"\n", virthwver) +
				fmt.Sprintf("displayName = \\\"%s\\\"\n", guestName) +
				fmt.Sprintf("numvcpus = \\\"%d\\\"\n", numvcpus) +
				fmt.Sprintf("memSize = \\\"%d\\\"\n", memsize) +
				fmt.Sprintf("guestOS = \\\"%s\\\"\n", guestos) +
				fmt.Sprintf("annotation = \\\"%s\\\"\n", notes) +
				fmt.Sprintf("floppy0.present = \\\"FALSE\\\"\n") +
				fmt.Sprintf("scsi0.present = \\\"TRUE\\\"\n") +
				fmt.Sprintf("scsi0.sharedBus = \\\"none\\\"\n") +
				fmt.Sprintf("scsi0.virtualDev = \\\"lsilogic\\\"\n") +
				fmt.Sprintf("pciBridge0.present = \\\"TRUE\\\"\n") +
				fmt.Sprintf("pciBridge4.present = \\\"TRUE\\\"\n") +
				fmt.Sprintf("pciBridge4.virtualDev = \\\"pcieRootPort\\\"\n") +
				fmt.Sprintf("pciBridge4.functions = \\\"8\\\"\n") +
				fmt.Sprintf("pciBridge5.present = \\\"TRUE\\\"\n") +
				fmt.Sprintf("pciBridge5.virtualDev = \\\"pcieRootPort\\\"\n") +
				fmt.Sprintf("pciBridge5.functions = \\\"8\\\"\n") +
				fmt.Sprintf("pciBridge6.present = \\\"TRUE\\\"\n") +
				fmt.Sprintf("pciBridge6.virtualDev = \\\"pcieRootPort\\\"\n") +
				fmt.Sprintf("pciBridge6.functions = \\\"8\\\"\n") +
				fmt.Sprintf("pciBridge7.present = \\\"TRUE\\\"\n") +
				fmt.Sprintf("pciBridge7.virtualDev = \\\"pcieRootPort\\\"\n") +
				fmt.Sprintf("pciBridge7.functions = \\\"8\\\"\n") +
				fmt.Sprintf("scsi0:0.present = \\\"TRUE\\\"\n") +
				fmt.Sprintf("scsi0:0.fileName = \\\"%s.vmdk\\\"\n", guestName) +
				fmt.Sprintf("scsi0:0.deviceType = \\\"scsi-hardDisk\\\"\n")
		if hasISO == true {
			vmxContent = vmxContent +
				fmt.Sprintf("ide1:0.present = \\\"TRUE\\\"\n") +
				fmt.Sprintf("ide1:0.fileName = \\\"emptyBackingString\\\"\n") +
				fmt.Sprintf("ide1:0.deviceType = \\\"atapi-cdrom\\\"\n") +
				fmt.Sprintf("ide1:0.startConnected = \\\"FALSE\\\"\n") +
				fmt.Sprintf("ide1:0.clientDevice = \\\"TRUE\\\"\n")
		} else {
			vmxContent = vmxContent +
				fmt.Sprintf("ide1:0.present = \\\"TRUE\\\"\n") +
				fmt.Sprintf("ide1:0.fileName = \\\"%s\\\"\n", isofilename) +
				fmt.Sprintf("ide1:0.deviceType = \\\"cdrom-image\\\"\n")
		}

		//
		//  Write vmx file to esxi host
		//
		log.Printf("[guestCREATE] New guest_name.vmx: %s\n", vmxContent)

		destVmxFile := fmt.Sprintf("%s/%s.vmx", fullPATH, guestName)

		remoteCmd = fmt.Sprintf("echo \"%s\" >%s", vmxContent, destVmxFile)
		vmxContent, err = runCommandOnHost(esxiSSHinfo, remoteCmd, "write guest_name.vmx file")

		//  Create boot disk (vmdk)
		remoteCmd = fmt.Sprintf("vmkfstools -c %sG -d %s %s/%s.vmdk", bootDiskSize, bootDiskType, fullPATH, guestName)
		_, err = runCommandOnHost(esxiSSHinfo, remoteCmd, "vmkfstools (make boot disk)")
		if err != nil {
			remoteCmd = fmt.Sprintf("rm -fr %s", fullPATH)
			stdout, _ = runCommandOnHost(esxiSSHinfo, remoteCmd, "cleanup guest path because of failed events")
			log.Printf("Failed to vmkfstools (make boot disk):%s\n", err.Error())
			return "", fmt.Errorf("Failed to vmkfstools (make boot disk):%s", err.Error())
		}

		poolID, err := getResourcePoolID(c, resourcePoolName)
		log.Println("[guestCREATE] DEBUG: " + poolID)
		if err != nil {
			log.Printf("Failed to use Resource Pool ID:%s\n", poolID)
			return "", fmt.Errorf("Failed to use Resource Pool ID:%s", poolID)
		}
		remoteCmd = fmt.Sprintf("vim-cmd solo/registervm %s %s %s", destVmxFile, guestName, poolID)
		_, err = runCommandOnHost(esxiSSHinfo, remoteCmd, "solo/registervm")
		if err != nil {
			log.Printf("Failed to register guest:%s\n", err.Error())
			remoteCmd = fmt.Sprintf("rm -fr %s", fullPATH)
			stdout, _ = runCommandOnHost(esxiSSHinfo, remoteCmd, "cleanup guest path because of failed events")
			return "", fmt.Errorf("Failed to register guest:%s", err.Error())
		}

	} else {
		//  Build VM by ovftool

		//  Check if source file exist.
		if !strings.HasPrefix(srcPath, "vi://") {
			if _, err := os.Stat(srcPath); os.IsNotExist(err) {
				return "", fmt.Errorf("File not found: %s", srcPath)
			}
		}

		//  Set params for ovftool
		if bootDiskType == "zeroedthick" {
			bootDiskType = "thick"
		}
		password := url.QueryEscape(c.esxiPassword)
		destPath := fmt.Sprintf("vi://%s:%s@%s/%s", c.esxiUserName, password, c.esxiHostName, resourcePoolName)

		networkParam := ""
		if (strings.HasSuffix(srcPath, ".ova") || strings.HasSuffix(srcPath, ".ovf")) && virtualNetworks[0][0] != "" {
			networkParam = " --network='" + virtualNetworks[0][0] + "'"
		}

		ovfCmd := fmt.Sprintf("ovftool --acceptAllEulas --noSSLVerify --X:useMacNaming=false "+
			"-dm=%s --name='%s' --overwrite -ds='%s' %s '%s' '%s'", bootDiskType, guestName, diskStore, networkParam, srcPath, destPath)

		if runtime.GOOS == "windows" {
			osShellCmd = "cmd.exe"
			osShellCmdOpt = "/c"

			ovfCmd = strings.Replace(ovfCmd, "'", "\"", -1)

			var ovfBat = "ovf_cmd.bat"

			_, err = os.Stat(ovfBat)

			// delete file if exists
			if os.IsExist(err) {
				err = os.Remove(ovfBat)
				if err != nil {
					return "", fmt.Errorf("Unable to delete %s: %s", ovfBat, err.Error())
				}
			}

			//  create new batch file
			file, err := os.Create(ovfBat)
			if err != nil {
				defer file.Close()
				return "", fmt.Errorf("Unable to create %s: %s", ovfBat, err.Error())
			}

			_, err = file.WriteString(ovfCmd)
			if err != nil {
				defer file.Close()
				return "", fmt.Errorf("Unable to write to %s: %s", ovfBat, err.Error())
			}

			err = file.Sync()
			defer file.Close()
			ovfCmd = ovfBat

		} else {
			osShellCmd = "/bin/bash"
			osShellCmdOpt = "-c"
		}

		//  Execute ovftool script (or batch) here.
		cmd := exec.Command(osShellCmd, osShellCmdOpt, ovfCmd)

		log.Printf("[guestCREATE] ovf_cmd: %s\n", ovfCmd)

		cmd.Stdout = &out
		err = cmd.Run()
		log.Printf("[guestCREATE] ovftool output: %q\n", out.String())

		if err != nil {
			log.Printf("Failed, There was an ovftool Error: %s\n%s\n", out.String(), err.Error())
			return "", fmt.Errorf("There was an ovftool Error: %s\n%s", out.String(), err.Error())
		}
	}

	// get VMID (by name)
	vmid, err = getGuestVMID(c, guestName)
	if err != nil {
		return "", err
	}

	//
	//  Grow boot disk to boot_disk_size
	//
	bootDiskVmdkPath, _ = getBootDiskPath(c, vmid)

	err = growVirtualDisk(c, bootDiskVmdkPath, bootDiskSize)
	if err != nil {
		return vmid, fmt.Errorf("Failed to grow boot disk")
	}

	//
	//  make updates to vmx file
	//
	err = updateVmx(c, vmid, true, memsize, numvcpus, virthwver, guestos, virtualNetworks, virtualDisks, notes, guestinfo)
	if err != nil {
		return vmid, err
	}

	return vmid, nil
}
