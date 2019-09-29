package esxi

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
)

// ValidateDiskStore checks that the requested disk store exists
func ValidateDiskStore(c *Config, diskStore string) error {
	esxiSSHinfo := SSHConnectionSettings{c.esxiHostName, c.esxiHostPort, c.esxiUserName, c.esxiPassword}
	log.Printf("[diskStoreValidate]\n")

	var remoteCmd, stdout string
	var err error

	//
	//  Check if Disk Store already exists
	//
	remoteCmd = fmt.Sprintf("esxcli storage filesystem list | grep '/vmfs/volumes/.*[VMFS|NFS]' | awk '{print $2}'")
	stdout, err = RunHostCommand(esxiSSHinfo, remoteCmd, "Get list of disk stores")
	if err != nil {
		return fmt.Errorf("Unable to get list of disk stores: %s", err)
	}
	log.Printf("1: Available Disk Stores: %s\n", strings.Replace(stdout, "\n", " ", -1))

	if strings.Contains(stdout, diskStore) == false {
		remoteCmd = fmt.Sprintf("esxcli storage filesystem rescan")
		_, _ = RunHostCommand(esxiSSHinfo, remoteCmd, "Refresh filesystems")

		remoteCmd = fmt.Sprintf("esxcli storage filesystem list | grep '/vmfs/volumes/.*[VMFS|NFS]' | awk '{print $2}'")
		stdout, err = RunHostCommand(esxiSSHinfo, remoteCmd, "Get list of disk stores")
		if err != nil {
			return fmt.Errorf("Unable to get list of disk stores: %s", err)
		}
		log.Printf("2: Available Disk Stores: %s\n", strings.Replace(stdout, "\n", " ", -1))

		if strings.Contains(stdout, diskStore) == false {
			return fmt.Errorf("Disk Store %s does not exist.\nAvailable Disk Stores: %s", diskStore, stdout)
		}
	}
	return nil
}

// CreateVirtualDisk creates the virtual disk on the host
func CreateVirtualDisk(c *Config, virtDiskDiskStore string, virtDiskDir string,
	virtDiskName string, virtDiskSize int, virtDiskType string) (string, error) {
	esxiSSHinfo := SSHConnectionSettings{c.esxiHostName, c.esxiHostPort, c.esxiUserName, c.esxiPassword}
	log.Println("[virtualDiskCREATE]")

	var virtDiskID, remoteCmd string
	var err error

	//
	//  Validate disk store exists
	//
	err = ValidateDiskStore(c, virtDiskDiskStore)
	if err != nil {
		return "", err
	}

	//
	//  Create dir if required
	//
	remoteCmd = fmt.Sprintf("mkdir -p \"/vmfs/volumes/%s/%s\"", virtDiskDiskStore, virtDiskDir)
	_, _ = RunHostCommand(esxiSSHinfo, remoteCmd, "create virtual disk dir")

	remoteCmd = fmt.Sprintf("ls -d \"/vmfs/volumes/%s/%s\"", virtDiskDiskStore, virtDiskDir)
	_, err = RunHostCommand(esxiSSHinfo, remoteCmd, "validate dir exists")
	if err != nil {
		return "", errors.New("Unable to create virtual_disk directory")
	}

	//
	//  virtdisk_id is just the full path name.
	//
	virtDiskID = fmt.Sprintf("/vmfs/volumes/%s/%s/%s", virtDiskDiskStore, virtDiskDir, virtDiskName)

	//
	//  Validate if it exists already
	//
	remoteCmd = fmt.Sprintf("ls -l \"%s\"", virtDiskID)
	_, err = RunHostCommand(esxiSSHinfo, remoteCmd, "validate disk store exists")
	if err == nil {
		log.Println("[virtualDiskCREATE]  Already exists.")
		return virtDiskID, err
	}

	remoteCmd = fmt.Sprintf("/bin/vmkfstools -c %dG -d %s \"%s\"", virtDiskSize,
		virtDiskType, virtDiskID)
	_, err = RunHostCommand(esxiSSHinfo, remoteCmd, "Create virtual_disk")
	if err != nil {
		return "", errors.New("Unable to create virtual_disk")
	}

	return virtDiskID, err
}

// GrowVirtualDisk grows the virtual disk to the intended size
func GrowVirtualDisk(c *Config, virtDiskID string, virtDiskSize string) error {
	esxiSSHinfo := SSHConnectionSettings{c.esxiHostName, c.esxiHostPort, c.esxiUserName, c.esxiPassword}
	log.Printf("[growVirtualDisk]\n")

	var newDiskSize int

	_, _, _, currentDiskSize, _, err := ReadVirtualDiskInfo(c, virtDiskID)

	newDiskSize, _ = strconv.Atoi(virtDiskSize)

	log.Printf("[growVirtualDisk] currentDiskSize:%d new_size:%d fullPATH: %s\n", currentDiskSize, newDiskSize, virtDiskID)

	if currentDiskSize < newDiskSize {
		remoteCmd := fmt.Sprintf("/bin/vmkfstools -X %dG \"%s\"", newDiskSize, virtDiskID)
		_, err := RunHostCommand(esxiSSHinfo, remoteCmd, "grow disk")
		if err != nil {
			return err
		}
	}

	return err
}

// ReadVirtualDiskInfo reads the virtual disk info from the host
func ReadVirtualDiskInfo(c *Config, virtDiskID string) (string, string, string, int, string, error) {
	esxiSSHinfo := SSHConnectionSettings{c.esxiHostName, c.esxiHostPort, c.esxiUserName, c.esxiPassword}
	log.Println("[virtualDiskREAD] Begin")

	var virtDiskDiskStore, virtDiskDir, virtDiskName string
	var virtDiskType, flatSize string
	var virtDiskSize int
	var flatSizei64 int64
	var s []string

	//  Split virtdisk_id into it's variables
	s = strings.Split(virtDiskID, "/")
	log.Printf("[virtualDiskREAD] len=%d cap=%d %v\n", len(s), cap(s), s)
	if len(s) < 6 {
		return "", "", "", 0, "", nil
	}
	virtDiskDiskStore = s[3]
	virtDiskDir = s[4]
	virtDiskName = s[5]

	// Test if virtual disk exists
	remoteCmd := fmt.Sprintf("test -s \"%s\"", virtDiskID)
	_, err := RunHostCommand(esxiSSHinfo, remoteCmd, "test if virtual disk exists")
	if err != nil {
		return "", "", "", 0, "", err
	}

	//  Get virtual disk flat size
	s = strings.Split(virtDiskName, ".")
	if len(s) < 2 {
		return "", "", "", 0, "", err
	}
	virtDiskNameFlat := fmt.Sprintf("%s-flat.%s", s[0], s[1])

	remoteCmd = fmt.Sprintf("ls -l \"/vmfs/volumes/%s/%s/%s\" | awk '{print $5}'",
		virtDiskDiskStore, virtDiskDir, virtDiskNameFlat)
	flatSize, err = RunHostCommand(esxiSSHinfo, remoteCmd, "Get size")
	if err != nil {
		return "", "", "", 0, "", err
	}
	flatSizei64, _ = strconv.ParseInt(flatSize, 10, 64)
	virtDiskSize = int(flatSizei64 / 1024 / 1024 / 1024)

	// Determine virtual disk type  (only works if Guest is powered off)
	remoteCmd = fmt.Sprintf("vmkfstools -t0 %s |grep -q 'VMFS Z- LVID:' && echo true", virtDiskID)
	isZeroedThick, _ := RunHostCommand(esxiSSHinfo, remoteCmd, "Get disk type.  Is zeroedthick.")

	remoteCmd = fmt.Sprintf("vmkfstools -t0 %s |grep -q 'VMFS -- LVID:' && echo true", virtDiskID)
	isEagerZeroedThick, _ := RunHostCommand(esxiSSHinfo, remoteCmd, "Get disk type.  Is eagerzeroedthick.")

	remoteCmd = fmt.Sprintf("vmkfstools -t0 %s |grep -q 'NOMP -- :' && echo true", virtDiskID)
	isThin, _ := RunHostCommand(esxiSSHinfo, remoteCmd, "Get disk type.  Is thin.")

	if isThin == "true" {
		virtDiskType = "thin"
	} else if isZeroedThick == "true" {
		virtDiskType = "zeroedthick"
	} else if isEagerZeroedThick == "true" {
		virtDiskType = "eagerzeroedthick"
	} else {
		virtDiskType = "Unknown"
	}

	// Return results
	return virtDiskDiskStore, virtDiskDir, virtDiskName, virtDiskSize, virtDiskType, err
}
