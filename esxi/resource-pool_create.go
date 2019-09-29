package esxi

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
)

func createResourcePoolResource(d *schema.ResourceData, m interface{}) error {
	c := m.(*Config)
	esxiSSHinfo := SSHConnectionSettings{c.esxiHostName, c.esxiHostPort, c.esxiUserName, c.esxiPassword}
	log.Println("[resourceRESOURCEPOOLCreate]")

	var remoteCmd string
	var poolID, parentPool string
	var err error

	resourcePoolName := d.Get("resource_pool_name").(string)
	cpuMin := d.Get("cpu_min").(int)
	cpuMinExpandable := d.Get("cpu_min_expandable").(string)
	cpuMax := d.Get("cpu_max").(int)
	cpuShares := strings.ToLower(d.Get("cpu_shares").(string))
	memMin := d.Get("mem_min").(int)
	memMinExpandable := d.Get("mem_min_expandable").(string)
	memMax := d.Get("mem_max").(int)
	memShares := strings.ToLower(d.Get("mem_shares").(string))
	parentPool = "Resources"

	if resourcePoolName == string('/') {
		return fmt.Errorf("Missing required resource_pool_name")
	}

	if resourcePoolName[0] == '/' {
		return fmt.Errorf("Resource Pool Name cannot start with /")
	}
	i := strings.LastIndex(resourcePoolName, "/")
	if i > 2 {
		parentPool = resourcePoolName[:i]
		resourcePoolName = resourcePoolName[i+1:]
	}

	//  Check if already exists
	stdout, _ := getResourcePoolID(c, resourcePoolName)
	if stdout != "" {
		d.SetId(stdout)
		return nil
	}

	cpuMinOpt := "--cpu-min=0"
	if cpuMin > 0 {
		cpuMinOpt = fmt.Sprintf("--cpu-min=%d", cpuMin)
	}

	cpuMinExpandableOpt := "--cpu-min-expandable=true"
	if cpuMinExpandable == "false" {
		cpuMinExpandableOpt = "--cpu-min-expandable=false"
	}

	cpuMaxOpt := ""
	if cpuMax > 0 {
		cpuMaxOpt = fmt.Sprintf("--cpu-max=%d", cpuMax)
	}

	cpuSharesOpt := "--cpu-shares=normal"
	if cpuShares == "low" || cpuShares == "high" {
		cpuSharesOpt = fmt.Sprintf("--cpu-shares=%s", cpuShares)
	} else {
		tmpVar, err := strconv.Atoi(cpuShares)
		if err == nil {
			cpuSharesOpt = fmt.Sprintf("--cpu-shares=%d", tmpVar)
		}
	}

	memMinOpt := "--mem-min=0"
	if memMin > 0 {
		memMinOpt = fmt.Sprintf("--mem-min=%d", memMin)
	}

	memMinExpandableOpt := "--mem-min-expandable=true"
	if memMinExpandable == "false" {
		memMinExpandableOpt = "--mem-min-expandable=false"
	}

	memMaxOpt := ""
	if memMax > 0 {
		memMaxOpt = fmt.Sprintf("--mem-max=%d", memMax)
	}

	memSharesOpt := "--mem-shares=normal"
	if memShares == "low" || memShares == "high" {
		memSharesOpt = fmt.Sprintf("--mem-shares=%s", memShares)
	} else {
		tmpVar, err := strconv.Atoi(memShares)
		if err == nil {
			memSharesOpt = fmt.Sprintf("--mem-shares=%d", tmpVar)
		}
	}

	parentPoolID, err := getResourcePoolID(c, parentPool)
	if err != nil {
		d.SetId("")
		return err
	}

	remoteCmd = fmt.Sprintf("vim-cmd hostsvc/rsrc/create %s %s %s %s %s %s %s %s %s %s",
		cpuMinOpt, cpuMinExpandableOpt, cpuMaxOpt, cpuSharesOpt,
		memMinOpt, memMinExpandableOpt, memMaxOpt, memSharesOpt, parentPoolID, resourcePoolName)

	_, err = runCommandOnHost(esxiSSHinfo, remoteCmd, "create resource pool")
	poolID, _ = getResourcePoolID(c, resourcePoolName)
	if err != nil {
		d.SetId("")
		return err
	}

	//  Set pool_id
	d.SetId(poolID)

	// Refresh
	resourcePoolName, cpuMin, cpuMinExpandable, cpuMax, cpuShares, memMin, memMinExpandable, memMax, memShares, err = readResourcePoolData(c, poolID)
	if err != nil {
		d.SetId("")
		return nil
	}

	d.Set("resource_pool_name", resourcePoolName)
	d.Set("cpu_min", cpuMin)
	d.Set("cpu_min_expandable", cpuMinExpandable)
	d.Set("cpu_max", cpuMax)
	d.Set("cpu_shares", cpuShares)
	d.Set("mem_min", memMin)
	d.Set("mem_min_expandable", memMinExpandable)
	d.Set("mem_max", memMax)
	d.Set("mem_shares", memShares)

	return nil
}
