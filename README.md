Terraform Provider
==================

- Website: https://www.terraform.io
- [![Gitter chat](https://badges.gitter.im/hashicorp-terraform/Lobby.png)](https://gitter.im/hashicorp-terraform/Lobby)
- Mailing list: [Google Groups](http://groups.google.com/group/terraform-tool)


Requirements
------------
-   [Terraform](https://www.terraform.io/downloads.html) 0.10.1+
-   [Go](https://golang.org/doc/install) 1.9 (to build the provider plugin)
-   [ovftool](https://www.vmware.com/support/developer/ovf/) from VMware.  NOTE: ovftool installer for windows doesn't put ovftool.exe in your path.  You will need to manually set your path.
-   You MUST enable ssh access on your ESXi hypervisor.
  * Google 'How to enable ssh access on esxi'
-   In general, you should know how to use terraform, esxi and some networking...
  * You will most likely need a DHCP server on your primary network if you are deploying VMs with public OVF/OVA/VMX images.  (Sources that have unconfigured primary interfaces.)
- The source OVF/OVA/VMX images must have open-vm-tools or vmware-tools installed to properly import an IPaddress.  (you need this to run provisioners)


Building The Provider
---------------------

You first must set your GOPATH.   If you are unsure, please review the documentation at.
>https://github.com/golang/go/wiki/SettingGOPATH


Clone repository to: `$GOPATH/src/github.com/terraform-providers/terraform-provider-esxi`

```sh

mkdir $HOME/go
export GOPATH="$HOME/go"

go get -u -v golang.org/x/crypto/ssh
go get -u -v github.com/hashicorp/terraform
go get -u -v github.com/josenk/terraform-provider-esxi

cd $GOPATH/src/github.com/josenk/terraform-provider-esxi
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags '-w -extldflags "-static"' -o terraform-provider-esxi_`cat version`

sudo cp terraform-provider-esxi_`cat version` /usr/local/bin
```

Terraform-provider-esxi plugin
==============================
* This is a Terraform plugin that adds a VMware ESXi provider support.  This allows Terraform to control and provision VMs directly on an ESXi hypervisor without a need for vCenter or VShpere.   ESXi hypervisor is a free download from VMware!
>https://www.vmware.com/go/get-free-esxi

* If you don't know terraform, I highly recommend you read through the introduction on the hashicorp website.
>https://www.terraform.io/intro/getting-started/install.html

* VMware Configuration Maximums tool.
>https://configmax.vmware.com/guest


What's New:
-----------
* Terraform can import existing Guest VMs, Virtual Disks & Resource pools by name. See wiki page for more info.
>https://github.com/josenk/terraform-provider-esxi/wiki/How-to-import
* Added support for GuestInfo.  (Thanks for the contribution silasb.)
  * This adds great provisioning options like Ignition and Cloud-Init!



Features and Compatibility
--------------------------
* Source image can be a clone of a VM or local vmx, ovf, ova file. This provider uses ovftool, so there should be a wide compatibility.
* Supports adding your VM to Resource Pools to partition CPU and memory usage from other VMs on your ESXi host.
* Terraform will Create, Destroy, Update & Read Resource Pools.
* Terraform will Create, Destroy, Update & Read Guest VMs.
* Terraform will Create, Destroy, Update & Read Extra Storage for Guests.


Vagrant vs Terraform.
---------------------
If you are using vagrant as a deployment tool (infa as code), you may want to consider a better tool.  Terraform.  Vagrant is better for development environments, while Terraform is better at managing infrastructure.  Please give my terraform plugin a try and give me some feedback.  What you're trying to do, what's missing, what works, what doesn't work, etc...
>https://www.vagrantup.com/intro/vs/terraform.html
>https://github.com/josenk/terraform-provider-esxi
>https://github.com/josenk/vagrant-vmware-esxi


Why this plugin?
----------------
Not everyone has vCenter, vSphere, expensive APIs...  These cost $$$.  ESXi is free!


How to install
--------------
Download and install Terraform on your local system using instructions from https://www.terraform.io/downloads.html.
Clone this plugin from github, build and place a copy of it in your path or current directory of your terraform project.  Or download pre-built binaries from https://github.com/josenk/terraform-provider-esxi/releases.


How to use and configure a main.tf file
---------------------------------------
1. cd SOMEDIR
2. `vi main.tf`  # Use the contents of this example main.tf as a template. Specify provider parameters to access your ESXi host.  Modify the resources for resource pools and guest vm.

```
provider "esxi" {
  esxi_hostname      = "esxi"
  esxi_hostport      = "22"
  esxi_username      = "root"
  esxi_password      = "MyPassword"
}

resource "esxi_guest" "vmtest" {
  guest_name         = "vmtest"
  disk_store         = "MyDiskStore"

  #
  #  Specify an existing guest to clone, an ovf source, or neither to build a bare-metal guest vm.
  #
  #clone_from_vm      = "Templates/centos7"
  #ovf_source        = "/local_path/centos-7.vmx"

  network_interfaces {
    virtual_network = "VM Network"
  }
  network_interfaces {
    virtual_network = "VM Network2"
  }
}
```

Basic usage
-----------
3. `terraform init`
4. `terraform plan`
5. `terraform apply`
6. `terraform show`
7. `terraform destroy`

Configuration reference
-----------------------
* provider "esxi"
  * esxi_hostname - Required
  * esxi_hostport - Optional - Default "22".
  * esxi_username - Optional - Default "root".
  * esxi_password - Required


* resource "esxi_resource_pool"
  * resource_pool_name - Required - The Resource Pool name.
  * cpu_min - Optional
  * cpu_min_expandable - Optional
  * cpu_max - Optional           
  * cpu_shares - Optional        
  * mem_min - Optional          
  * mem_min_expandable - Optional
  * mem_max - Optional           
  * mem_shares - Optional


* resource "esxi_virtual_disk"
  * virtual_disk_disk_store - Required - esxi Disk Store where guest vm will be created.
  * virtual_disk_dir - Required - Disk dir.
  * virtual_disk_name - Optional - Virtual Disk Name. A random virtual disk name will be generated if nil.
  * virtual_disk_size - Optional - Virtual Disk size in GB. Default 1GB.
  * virtual_disk_type - Optional - Virtual Disk type.  (thin, zeroedthick or eagerzeroedthick) Default 'thin'.


* resource "esxi_guest"
  * guest_name - Required - The Guest name.
  * ip_address - Computed - The IP address reported by VMware tools.
  * boot_disk_type - Optional - Guest boot disk type. Default 'thin'.  Available thin, zeroedthick, eagerzeroedthick.
  * boot_disk_size - Optional - Specify boot disk size or grow cloned vm to this size.
  * guestos - Optional - Default will be taken from cloned source.
  * clone_from_vm - Source vm to clone. Mutually exclusive with ovf_source option.     
  * ovf_source - ovf files to use as a source. Mutually exclusive with clone_from_vm option.      
  * disk_store - Required - esxi Disk Store where guest vm will be created.    
  * resource_pool_name - Optional - Any existing or terraform managed resource pool name. - Default "/".      
  * memsize - Optional - Memory size in MB.  (ie, 1024 == 1GB). See esxi documentation for limits. - Default 512 or default taken from cloned source.
  * numvcpus - Optional - Number of virtual cpus.  See esxi documentation for limits. - Default 1 or default taken from cloned source.
  * virthwver - Optional - esxi guest virtual HW version.  See esxi documentation for compatible values. - Default 8 or taken from cloned source.
  * network_interfaces - Array of upto 10 network interfaces.
    * virtual_network - Required for each Guest NIC - This is the esxi virtual network name configured on esxi host.
    * mac_address - Optional -  If not set, mac_address will be generated by esxi.
    * nic_type - Optional - See esxi documentation for compatibility list. - Default "e1000" or taken from cloned source.
  * virtual_disks - Optional - Array of additional storage to be added to the guest.
    * virtual_disk_id - Required - virtual_disk.id from esxi_virtual_disk resource.
    * slot - Required - SCSI_Ctrl:SCSI_id.  Range  '0:1' to '3:15'.  SCSI_id 7 is not allowed.
  * power - Optional - on, off.
  * guest_startup_timeout - Optional - The amount of guest uptime, in seconds, to wait for an available IP address on this virtual machine.
  * guest_shutdown_timeout - Optional - The amount of time, in seconds, to wait for a graceful shutdown before doing a forced power off.
  * notes - Optional - The Guest notes (annotation).
  * guestinfo - Optional - The Guestinfo root
    * metadata - Optional - A JSON string containing the cloud-init metadata.
    * metadata.encoding - Optional - The encoding type for guestinfo.metadata. (base64 or gzip+base64)
    * userdata - Optional - A YAML document containing the cloud-init user data.
    * userdata.encoding - Optional - The encoding type for guestinfo.userdata. (base64 or gzip+base64)
    * vendordata - Optional - A YAML document containing the cloud-init vendor data.
    * vendordata.encoding - Optional - The encoding type for guestinfo.vendordata (base64 or gzip+base64)


Known issues with vmware_esxi
-----------------------------
* terraform import cannot import the guest disk type (thick, thin, etc) if the VM is powered on and cannot import the guest ip_address if it's powered off.
* Only numvcpus are supported.   numcores is not supported.
* Doesn't support CDrom or floppy.
* Doesn't support Shared bus Interfaces, or Shared disks


Version History
---------------
* 1.5.0 Support for Terraform 0.12, migrated examples to 0.12 format. Support to modify virtual_network & nic_type.  Windows fixes.
* 1.4.3 Fix virtdisk count. Fixes to support Terraform 0.12
* 1.4.2 Support 10 nics, more README changes
* 1.4.1 Fix README build instructions, static binaries, update guest types
* 1.4.0 Add GuestInfo (Cloud-init, Ignition!).   Fix, allow esxi passwords with special characters.
* 1.3.0 Add support to Update storage attachments.
* 1.2.2 fix guest_update power, boot_disk_type defaults, README, windows support
* 1.2.1 Fix ssh connection retries.
* 1.2.0 Add support for notes (annotation)
* 1.1.1 Fix, unable to provision ova sources.  go fmt.
* 1.1.0 Add Import support.
* 1.0.2 Switch authentication method to Keyboard Interactive.  Read disk_type (thin, thick, etc)
* 1.0.1 Validate DiskStores and refresh
* 1.0.0 First Major release
* 0.1.2 Add ability to manage existing Guest VMs.  A lot of code cleanup, various fixes, more validation.
* 0.1.0 Add virtual_disk resource.
* 0.0.8 Add virthwver.
* 0.0.7 build vmx from scratch if no source is specified
* 0.0.6 Add power resource.
* 0.0.5 Add network_interfaces resource.
* 0.0.4 Add more stuff.
* 0.0.3 Add memory and numvcpus resource.
      Add support to update some guests params.
* 0.0.2 Add Resource Pool resource.
* 0.0.1 Init release
