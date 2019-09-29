package esxi

import (
	"bytes"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// CreateGuest creates a guest VM on the host
func CreateGuest(c *Config, guestName string, diskStore string,
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
	err = ValidateDiskStore(c, diskStore)
	if err != nil {
		return "", err
	}

	//
	//  Check if guest already exists
	//
	// get VMID (by name)
	vmid, err = GetGuestVMID(c, guestName)

	if vmid != "" {
		// We don't need to create the VM.   It already exists.
		fmt.Printf("[guestCREATE] guest %s already exists vmid: %s\n", guestName, stdout)

		//
		//   Power off guest if it's powered on.
		//
		currentpowerstate := GetGuestPowerState(c, vmid)
		if currentpowerstate == "on" || currentpowerstate == "suspended" {
			_, err = PowerOffGuest(c, vmid, guestShutdownTimeout)
			if err != nil {
				return "", fmt.Errorf("Failed to power off existing guest. vmid:%s", vmid)
			}
		}

	} else if srcPath == "none" {

		// check if path already exists.
		fullPATH := fmt.Sprintf("\"/vmfs/volumes/%s/%s\"", diskStore, guestName)
		bootDiskVmdkPath = fmt.Sprintf("\"/vmfs/volumes/%s/%s/%s.vmdk\"", diskStore, guestName, guestName)
		remoteCmd = fmt.Sprintf("ls -d %s", fullPATH)
		stdout, _ = RunHostCommand(esxiSSHinfo, remoteCmd, "check if guest path already exists.")
		if stdout == fullPATH {
			fmt.Printf("Error: Guest path already exists. fullPATH:%s\n", fullPATH)
			return "", fmt.Errorf("Guest path already exists. fullPATH:%s", fullPATH)
		}

		remoteCmd = fmt.Sprintf("mkdir %s", fullPATH)
		stdout, err = RunHostCommand(esxiSSHinfo, remoteCmd, "create guest path")
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
		vmxContent, err = RunHostCommand(esxiSSHinfo, remoteCmd, "write guest_name.vmx file")

		//  Create boot disk (vmdk)
		remoteCmd = fmt.Sprintf("vmkfstools -c %sG -d %s %s/%s.vmdk", bootDiskSize, bootDiskType, fullPATH, guestName)
		_, err = RunHostCommand(esxiSSHinfo, remoteCmd, "vmkfstools (make boot disk)")
		if err != nil {
			remoteCmd = fmt.Sprintf("rm -fr %s", fullPATH)
			stdout, _ = RunHostCommand(esxiSSHinfo, remoteCmd, "cleanup guest path because of failed events")
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
		_, err = RunHostCommand(esxiSSHinfo, remoteCmd, "solo/registervm")
		if err != nil {
			log.Printf("Failed to register guest:%s\n", err.Error())
			remoteCmd = fmt.Sprintf("rm -fr %s", fullPATH)
			stdout, _ = RunHostCommand(esxiSSHinfo, remoteCmd, "cleanup guest path because of failed events")
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
	vmid, err = GetGuestVMID(c, guestName)
	if err != nil {
		return "", err
	}

	//
	//  Grow boot disk to boot_disk_size
	//
	bootDiskVmdkPath, _ = GetBootDiskPath(c, vmid)

	err = GrowVirtualDisk(c, bootDiskVmdkPath, bootDiskSize)
	if err != nil {
		return vmid, fmt.Errorf("Failed to grow boot disk")
	}

	//
	//  make updates to vmx file
	//
	err = UpdateVmx(c, vmid, true, memsize, numvcpus, virthwver, guestos, virtualNetworks, virtualDisks, notes, guestinfo)
	if err != nil {
		return vmid, err
	}

	return vmid, nil
}
