package esxi

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
)

// getResourcePoolID checks if resource pool exists (by name )and return it's Pool ID.
func getResourcePoolID(c *Config, resourcePoolName string) (string, error) {
	esxiSSHinfo := SSHConnectionSettings{c.esxiHostName, c.esxiHostPort, c.esxiUserName, c.esxiPassword}
	log.Printf("[getPoolID]\n")

	if resourcePoolName == "/" || resourcePoolName == "Resources" {
		return "ha-root-pool", nil
	}

	result := strings.Split(resourcePoolName, "/")
	resourcePoolName = result[len(result)-1]

	r := strings.NewReplacer("objID>", "", "</objID", "")
	remoteCmd := fmt.Sprintf("grep -A1 '<name>%s</name>' /etc/vmware/hostd/pools.xml | grep -o objID.*objID | tail -1", resourcePoolName)
	stdout, err := RunHostCommand(esxiSSHinfo, remoteCmd, "get existing resource pool id")
	if err == nil {
		stdout = r.Replace(stdout)
		return stdout, err
	}

	log.Printf("[getPoolID] Failed get existing resource pool id: %s\n", stdout)
	return "", err

}

// getResourcePoolName checks if Pool exists (by id)and return it's Pool name.
func getResourcePoolName(c *Config, resourcePoolID string) (string, error) {
	esxiSSHinfo := SSHConnectionSettings{c.esxiHostName, c.esxiHostPort, c.esxiUserName, c.esxiPassword}
	log.Printf("[getPoolNAME]\n")

	var resourcePoolName, fullResourcePoolName string

	fullResourcePoolName = ""

	if resourcePoolID == "ha-root-pool" {
		return "/", nil
	}

	// Get full Resource Pool Path
	remoteCmd := fmt.Sprintf("grep -A1 '<objID>%s</objID>' /etc/vmware/hostd/pools.xml | grep '<path>'", resourcePoolID)
	stdout, err := RunHostCommand(esxiSSHinfo, remoteCmd, "get resource pool path")
	if err != nil {
		log.Printf("[getPoolNAME] Failed get resource pool PATH: %s\n", stdout)
		return "", err
	}

	re := regexp.MustCompile(`[/<>\n]`)
	result := re.Split(stdout, -1)

	for i := range result {

		resourcePoolName = ""
		if result[i] != "path" && result[i] != "host" && result[i] != "user" && result[i] != "" {

			r := strings.NewReplacer("name>", "", "</name", "")
			remoteCmd := fmt.Sprintf("grep -B1 '<objID>%s</objID>' /etc/vmware/hostd/pools.xml | grep -o name.*name", result[i])
			stdout, _ := RunHostCommand(esxiSSHinfo, remoteCmd, "get resource pool name")
			resourcePoolName = r.Replace(stdout)

			if resourcePoolName != "" {
				if result[i] == resourcePoolID {
					fullResourcePoolName = fullResourcePoolName + resourcePoolName
				} else {
					fullResourcePoolName = fullResourcePoolName + resourcePoolName + "/"
				}
			}
		}
	}

	return fullResourcePoolName, nil
}

func readResourcePoolData(c *Config, poolID string) (string, int, string, int, string, int, string, int, string, error) {
	esxiSSHinfo := SSHConnectionSettings{c.esxiHostName, c.esxiHostPort, c.esxiUserName, c.esxiPassword}
	log.Println("[resourcePoolRead]")

	var remoteCmd, stdout, cpuShares, memShares string
	var cpuMin, cpuMax, memMin, memMax, tmpvar int
	var cpuMinExpandable, memMinExpandable string
	var err error

	remoteCmd = fmt.Sprintf("vim-cmd hostsvc/rsrc/pool_config_get %s", poolID)
	stdout, err = RunHostCommand(esxiSSHinfo, remoteCmd, "resource pool_config_get")

	if strings.Contains(stdout, "deleted") == true {
		log.Printf("[resourcePoolRead] Already deleted: %s\n", err)
		return "", 0, "", 0, "", 0, "", 0, "", nil
	}
	if err != nil {
		log.Printf("[resourcePoolRead] Failed to get %s: %s\n", "resource pool_config_get", err)
		return "", 0, "", 0, "", 0, "", 0, "", errors.New("Failed to get Resource Pool config")
	}

	isCPUFlag := true

	scanner := bufio.NewScanner(strings.NewReader(stdout))
	for scanner.Scan() {
		switch {
		case strings.Contains(scanner.Text(), "memoryAllocation = "):
			isCPUFlag = false

		case strings.Contains(scanner.Text(), "reservation = "):
			r, _ := regexp.Compile("[0-9]+")
			if isCPUFlag == true {
				cpuMin, _ = strconv.Atoi(r.FindString(scanner.Text()))
			} else {
				memMin, _ = strconv.Atoi(r.FindString(scanner.Text()))
			}

		case strings.Contains(scanner.Text(), "expandableReservation = "):
			r, _ := regexp.Compile("(true|false)")
			if isCPUFlag == true {
				cpuMinExpandable = r.FindString(scanner.Text())
			} else {
				memMinExpandable = r.FindString(scanner.Text())
			}

		case strings.Contains(scanner.Text(), "limit = "):
			r, _ := regexp.Compile("-?[0-9]+")
			tmpvar, _ = strconv.Atoi(r.FindString(scanner.Text()))
			if tmpvar < 0 {
				tmpvar = 0
			}
			if isCPUFlag == true {
				cpuMax = tmpvar
			} else {
				memMax = tmpvar
			}

		case strings.Contains(scanner.Text(), "shares = "):
			r, _ := regexp.Compile("[0-9]+")
			if isCPUFlag == true {
				cpuShares = r.FindString(scanner.Text())
			} else {
				memShares = r.FindString(scanner.Text())
			}

		case strings.Contains(scanner.Text(), "level = "):
			r, _ := regexp.Compile("(low|high|normal)")
			if r.FindString(scanner.Text()) != "" {
				if isCPUFlag == true {
					cpuShares = r.FindString(scanner.Text())
				} else {
					memShares = r.FindString(scanner.Text())
				}
			}
		}
	}

	resourcePoolName, err := getResourcePoolName(c, poolID)
	if err != nil {
		log.Printf("[resourcePoolRead] Failed to get Resource Pool name: %s\n", err)
		return "", 0, "", 0, "", 0, "", 0, "", errors.New("Failed to get Resource Pool name")
	}

	return resourcePoolName, cpuMin, cpuMinExpandable, cpuMax, cpuShares,
		memMin, memMinExpandable, memMax, memShares, nil
}
