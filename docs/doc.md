<br>
<h3 id="settings.gardener.cloud/v1.AzureProviderSpec">
<b>AzureProviderSpec</b>
</h3>
<p>
<p>AzureProviderSpec is the provider specific configuration to use during node creation
on Azure.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
settings.gardener.cloud/v1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>AzureProviderSpec</code></td>
</tr>
<tr>
<td>
<code>location</code></br>
<em>
string
</em>
</td>
<td>
<p>Region in which virtual machine would be hosted.</p>
</td>
</tr>
<tr>
<td>
<code>tags</code></br>
<em>
map[string]string
</em>
</td>
<td>
<p>Identifier tags for virtual machines.</p>
</td>
</tr>
<tr>
<td>
<code>properties</code></br>
<em>
<a href="#settings.gardener.cloud/v1.AzureVirtualMachineProperties">
AzureVirtualMachineProperties
</a>
</em>
</td>
<td>
<p>AzureVirtualMachineProperties describes the properties of a Virtual Machine.</p>
</td>
</tr>
<tr>
<td>
<code>resourceGroup</code></br>
<em>
string
</em>
</td>
<td>
<p>Name of the Azure resource group</p>
</td>
</tr>
<tr>
<td>
<code>subnetInfo</code></br>
<em>
<a href="#settings.gardener.cloud/v1.AzureSubnetInfo">
AzureSubnetInfo
</a>
</em>
</td>
<td>
<p>AzureSubnetInfo is the information containing the subnet details</p>
</td>
</tr>
</tbody>
</table>
<br>
<h3 id="settings.gardener.cloud/v1.AzureDataDisk">
<b>AzureDataDisk</b>
</h3>
<p>
(<em>Appears on:</em>
<a href="#settings.gardener.cloud/v1.AzureStorageProfile">AzureStorageProfile</a>)
</p>
<p>
<p>AzureDataDisk Specifies the parameters that are used to add a data disk to a virtual machine.
For more information about disks, see <a href="https://docs.microsoft.com/en-us/azure/virtual-machines/managed-disks-overview">About disks and VHDs for Azure virtual machines</a>.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>The disk name.</p>
</td>
</tr>
<tr>
<td>
<code>lun</code></br>
<em>
*int32
</em>
</td>
<td>
<p>Specifies the logical unit number of the data disk. This value is used to identify data disks within the VM
and therefore must be unique for each data disk attached to a VM.</p>
</td>
</tr>
<tr>
<td>
<code>caching</code></br>
<em>
string
</em>
</td>
<td>
<p>Specifies the caching requirements.</p>
<p>Possible values are: <em>None</em>, <em>ReadOnly</em>, <em>ReadWrite</em></p>
<p>Default: <em>None</em> for Standard storage. <em>ReadOnly</em> for Premium storage</p>
</td>
</tr>
<tr>
<td>
<code>storageAccountType</code></br>
<em>
string
</em>
</td>
<td>
<p>Specifies the storage account type for the managed disk.
NOTE: UltraSSD_LRS can only be used with data disks, it cannot be used with OS Disk.</p>
</td>
</tr>
<tr>
<td>
<code>diskSizeGB</code></br>
<em>
int32
</em>
</td>
<td>
<p>Specifies the size of an empty data disk in gigabytes. This element can be used to overwrite the size of the disk in a virtual machine image.</p>
<p>This value cannot be larger than 1023 GB</p>
</td>
</tr>
</tbody>
</table>
<br>
<h3 id="settings.gardener.cloud/v1.AzureHardwareProfile">
<b>AzureHardwareProfile</b>
</h3>
<p>
(<em>Appears on:</em>
<a href="#settings.gardener.cloud/v1.AzureVirtualMachineProperties">AzureVirtualMachineProperties</a>)
</p>
<p>
<p>AzureHardwareProfile is specifies the hardware settings for the virtual machine.
Refer github.com/Azure/azure-sdk-for-go/arm/compute/models.go for VMSizes</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>vmSize</code></br>
<em>
string
</em>
</td>
<td>
<p>Specifies the size of the virtual machine. The enum data type is currently
deprecated and will be removed by December 23rd 2023. Recommended way to get
the list of available sizes is using these APIs:</p>
<ul>
<li>List all available virtual machine sizes in an availability set</li>
<li>List all available virtual machine sizes in a region</li>
<li>List all available virtual machine sizes for resizing.</li>
</ul>
<p>The available VM sizes depend on region and availability set.</p>
</td>
</tr>
</tbody>
</table>
<br>
<h3 id="settings.gardener.cloud/v1.AzureImageReference">
<b>AzureImageReference</b>
</h3>
<p>
(<em>Appears on:</em>
<a href="#settings.gardener.cloud/v1.AzureStorageProfile">AzureStorageProfile</a>)
</p>
<p>
<p>AzureImageReference specifies information about the image to use. You can specify information about platform images,
marketplace images, or virtual machine images. This element is required when you want to use a platform image,
marketplace image, or virtual machine image, but is not used in other creation operations.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>id</code></br>
<em>
string
</em>
</td>
<td>
<p>Resource Id</p>
</td>
</tr>
<tr>
<td>
<code>urn</code></br>
<em>
*string
</em>
</td>
<td>
<p>Uniform Resource Name of the OS image to be used , it has the format &lsquo;publisher:offer:sku:version&rsquo;</p>
</td>
</tr>
</tbody>
</table>
<br>
<h3 id="settings.gardener.cloud/v1.AzureLinuxConfiguration">
<b>AzureLinuxConfiguration</b>
</h3>
<p>
(<em>Appears on:</em>
<a href="#settings.gardener.cloud/v1.AzureOSProfile">AzureOSProfile</a>)
</p>
<p>
<p>AzureLinuxConfiguration is specifies the Linux operating system settings on the virtual machine.</p>
<p>For a list of supported Linux distributions, see <a href="https://docs.microsoft.com/azure/virtual-machines/virtual-machines-linux-endorsed-distros?toc=%2fazure%2fvirtual-machines%2flinux%2ftoc.json">Linux on Azure-Endorsed
Distributions</a></p>
<p>For running non-endorsed distributions, see <a href="https://docs.microsoft.com/azure/virtual-machines/virtual-machines-linux-create-upload-generic?toc=%2fazure%2fvirtual-machines%2flinux%2ftoc.json">Information for Non-Endorsed
Distributions</a>.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>disablePasswordAuthentication</code></br>
<em>
bool
</em>
</td>
<td>
<p>Specifies whether password authentication should be disabled.</p>
</td>
</tr>
<tr>
<td>
<code>ssh</code></br>
<em>
<a href="#settings.gardener.cloud/v1.AzureSSHConfiguration">
AzureSSHConfiguration
</a>
</em>
</td>
<td>
<p>Specifies the ssh key configuration for a Linux OS.</p>
</td>
</tr>
</tbody>
</table>
<br>
<h3 id="settings.gardener.cloud/v1.AzureMachineSetConfig">
<b>AzureMachineSetConfig</b>
</h3>
<p>
(<em>Appears on:</em>
<a href="#settings.gardener.cloud/v1.AzureVirtualMachineProperties">AzureVirtualMachineProperties</a>)
</p>
<p>
<p>AzureMachineSetConfig contains the information about the associated machineSet.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>id</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<br>
<h3 id="settings.gardener.cloud/v1.AzureManagedDiskParameters">
<b>AzureManagedDiskParameters</b>
</h3>
<p>
(<em>Appears on:</em>
<a href="#settings.gardener.cloud/v1.AzureOSDisk">AzureOSDisk</a>)
</p>
<p>
<p>AzureManagedDiskParameters is the parameters of a managed disk.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>id</code></br>
<em>
string
</em>
</td>
<td>
<p>Resource Id</p>
</td>
</tr>
<tr>
<td>
<code>storageAccountType</code></br>
<em>
string
</em>
</td>
<td>
<p>Specifies the storage account type for the managed disk.
NOTE: UltraSSD_LRS can only be used with data disks, it cannot be used with OS Disk.</p>
</td>
</tr>
</tbody>
</table>
<br>
<h3 id="settings.gardener.cloud/v1.AzureNetworkInterfaceReference">
<b>AzureNetworkInterfaceReference</b>
</h3>
<p>
(<em>Appears on:</em>
<a href="#settings.gardener.cloud/v1.AzureNetworkProfile">AzureNetworkProfile</a>)
</p>
<p>
<p>AzureNetworkInterfaceReference specifies the network interfaces of the virtual machine.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>id</code></br>
<em>
string
</em>
</td>
<td>
<p>Resource Id</p>
</td>
</tr>
<tr>
<td>
<code>properties</code></br>
<em>
<a href="#settings.gardener.cloud/v1.AzureNetworkInterfaceReferenceProperties">
AzureNetworkInterfaceReferenceProperties
</a>
</em>
</td>
<td>
<p>Specifies the primary network interface in case the virtual machine has
more than 1 network interface.</p>
</td>
</tr>
</tbody>
</table>
<br>
<h3 id="settings.gardener.cloud/v1.AzureNetworkInterfaceReferenceProperties">
<b>AzureNetworkInterfaceReferenceProperties</b>
</h3>
<p>
(<em>Appears on:</em>
<a href="#settings.gardener.cloud/v1.AzureNetworkInterfaceReference">AzureNetworkInterfaceReference</a>)
</p>
<p>
<p>AzureNetworkInterfaceReferenceProperties is describes a network interface
reference properties.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>primary</code></br>
<em>
bool
</em>
</td>
<td>
<p>Specifies the primary network interface in case the virtual machine
has more than 1 network interface.</p>
</td>
</tr>
</tbody>
</table>
<br>
<h3 id="settings.gardener.cloud/v1.AzureNetworkProfile">
<b>AzureNetworkProfile</b>
</h3>
<p>
(<em>Appears on:</em>
<a href="#settings.gardener.cloud/v1.AzureVirtualMachineProperties">AzureVirtualMachineProperties</a>)
</p>
<p>
<p>AzureNetworkProfile specifies the network interfaces of the virtual machine.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>networkInterfaces</code></br>
<em>
<a href="#settings.gardener.cloud/v1.AzureNetworkInterfaceReference">
AzureNetworkInterfaceReference
</a>
</em>
</td>
<td>
<p>Specifies the network interfaces of the virtual machine.</p>
</td>
</tr>
<tr>
<td>
<code>acceleratedNetworking</code></br>
<em>
*bool
</em>
</td>
<td>
<p>Specifies if the acceleration is enabled in network.</p>
</td>
</tr>
</tbody>
</table>
<br>
<h3 id="settings.gardener.cloud/v1.AzureOSDisk">
<b>AzureOSDisk</b>
</h3>
<p>
(<em>Appears on:</em>
<a href="#settings.gardener.cloud/v1.AzureStorageProfile">AzureStorageProfile</a>)
</p>
<p>
<p>AzureOSDisk specifies information about the operating system disk used by the virtual machine.
For more information about disks, see <a href="https://docs.microsoft.com/azure/virtual-machines/virtual-machines-windows-about-disks-vhds?toc=%2fazure%2fvirtual-machines%2fwindows%2ftoc.json">About disks and VHDs for Azure virtual
machines</a>.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>The disk name.</p>
</td>
</tr>
<tr>
<td>
<code>caching</code></br>
<em>
string
</em>
</td>
<td>
<p>Specifies the caching requirements.
Possible values are: <em>None</em>, <em>ReadOnly</em> or <em>ReadWrite</em>
Default: <em>None</em> for Standard storage. <em>ReadOnly</em> for Premium storage.</p>
</td>
</tr>
<tr>
<td>
<code>managedDisk</code></br>
<em>
<a href="#settings.gardener.cloud/v1.AzureManagedDiskParameters">
AzureManagedDiskParameters
</a>
</em>
</td>
<td>
<p>The managed disk parameters.</p>
</td>
</tr>
<tr>
<td>
<code>diskSizeGB</code></br>
<em>
int32
</em>
</td>
<td>
<p>Specifies the size of an empty data disk in gigabytes. This element can be used to
overwrite the size of the disk in a virtual machine image.</p>
<p>This value cannot be larger than 1023 GB.</p>
</td>
</tr>
<tr>
<td>
<code>createOption</code></br>
<em>
string
</em>
</td>
<td>
<p>Specifies how the virtual machine should be created.</p>
<p>Possible values are:</p>
<p><strong>Attach</strong> \u2013 This value is used when you are using a specialized disk to create the virtual machine.</p>
<p><strong>FromImage</strong> \u2013 This value is used when you are using an image to create the virtual machine. If you
are using a platform image, you also use the imageReference element described above. If you are using a
marketplace image, you also use the plan element previously described.</p>
</td>
</tr>
</tbody>
</table>
<br>
<h3 id="settings.gardener.cloud/v1.AzureOSProfile">
<b>AzureOSProfile</b>
</h3>
<p>
(<em>Appears on:</em>
<a href="#settings.gardener.cloud/v1.AzureVirtualMachineProperties">AzureVirtualMachineProperties</a>)
</p>
<p>
<p>AzureOSProfile specifies the operating system settings for the virtual machine.
Some of the settings cannot be changed once VM is provisioned. For more details
see <a href="https://docs.microsoft.com/en-us/rest/api/compute/virtual-machines/create-or-update#osprofile">documentation on osProfile in Azure Virtual Machines</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>computerName</code></br>
<em>
string
</em>
</td>
<td>
<p>Specifies the host OS name of the virtual machine. This name cannot be updated after the VM is created.</p>
<p><strong>Max-length (Windows)</strong>: 15 characters</p>
<p><strong>Max-length (Linux)</strong>: 64 characters.</p>
<p>For naming conventions and restrictions see <a href="https://docs.microsoft.com/en-us/azure/azure-resource-manager/management/resource-name-rules">Azure infrastructure services implementation
guidelines</a>.</p>
</td>
</tr>
<tr>
<td>
<code>adminUsername</code></br>
<em>
string
</em>
</td>
<td>
<p><strong>Windows-only restriction</strong>: Cannot end in &ldquo;.&rdquo;</p>
<p><strong>Disallowed values</strong>: &ldquo;administrator&rdquo;, &ldquo;admin&rdquo;, &ldquo;user&rdquo;, &ldquo;user1&rdquo;, &ldquo;test&rdquo;, &ldquo;user2&rdquo;, &ldquo;test1&rdquo;, &ldquo;user3&rdquo;, &ldquo;admin1&rdquo;,
&ldquo;1&rdquo;, &ldquo;123&rdquo;, &ldquo;a&rdquo;, &ldquo;actuser&rdquo;, &ldquo;adm&rdquo;, &ldquo;admin2&rdquo;, &ldquo;aspnet&rdquo;, &ldquo;backup&rdquo;, &ldquo;console&rdquo;, &ldquo;david&rdquo;, &ldquo;guest&rdquo;, &ldquo;john&rdquo;, &ldquo;owner&rdquo;,
&ldquo;root&rdquo;, &ldquo;server&rdquo;, &ldquo;sql&rdquo;, &ldquo;support&rdquo;, &ldquo;support_388945a0&rdquo;, &ldquo;sys&rdquo;, &ldquo;test2&rdquo;, &ldquo;test3&rdquo;, &ldquo;user4&rdquo;, &ldquo;user5&rdquo;.</p>
<p><strong>Minimum-length (Linux)</strong>: 1 character</p>
<p><strong>Max-length (Linux)</strong>: 64 characters</p>
<p><strong>Max-length (Windows)</strong>: 20 characters.</p>
</td>
</tr>
<tr>
<td>
<code>adminPassword</code></br>
<em>
string
</em>
</td>
<td>
<p>Specifies the password of the administrator account.</p>
<p><strong>Minimum-length (Windows)</strong>: 8 characters</p>
<p><strong>Minimum-length (Linux)</strong>: 6 characters</p>
<p><strong>Max-length (Windows)</strong>: 123 characters</p>
<p><strong>Max-length (Linux)</strong>: 72 characters</p>
<p><strong>Complexity requirements</strong>: 3 out of 4 conditions below need to be fulfilled
Has lower characters
Has upper characters
Has a digit
Has a special character (Regex match [\W_])</p>
<p><strong>Disallowed values</strong>: &ldquo;abc@123&rdquo;, &ldquo;P@$$w0rd&rdquo;, &ldquo;P@ssw0rd&rdquo;, &ldquo;P@ssword123&rdquo;, &ldquo;Pa$$word&rdquo;, &ldquo;pass@word1&rdquo;,
&ldquo;Password!&rdquo;, &ldquo;Password1&rdquo;, &ldquo;Password22&rdquo;, &ldquo;iloveyou!&rdquo;</p>
<p>For resetting the password, see <a href="https://docs.microsoft.com/en-us/troubleshoot/azure/virtual-machines/reset-rdp">How to reset the Remote Desktop service or its login password in a Windows
VM</a></p>
<p>For resetting root password, see <a href="https://docs.microsoft.com/en-us/troubleshoot/azure/virtual-machines/troubleshoot-ssh-connection">Manage users, SSH, and check or repair disks on Azure Linux VMs using the
VMAccess Extension</a></p>
</td>
</tr>
<tr>
<td>
<code>customData</code></br>
<em>
string
</em>
</td>
<td>
<p>This property cannot be updated after the VM is created.</p>
<p>customData is passed to the VM to be saved as a file, for more information see <a href="https://azure.microsoft.com/blog/custom-data-and-cloud-init-on-windows-azure/">Custom Data on Azure
VMs</a></p>
<p>For using cloud-init for your Linux VM, see <a href="https://docs.microsoft.com/en-us/azure/virtual-machines/linux/using-cloud-init">Using cloud-init to customize a Linux VM during
creation</a></p>
</td>
</tr>
<tr>
<td>
<code>linuxConfiguration</code></br>
<em>
<a href="#settings.gardener.cloud/v1.AzureLinuxConfiguration">
AzureLinuxConfiguration
</a>
</em>
</td>
<td>
<p>For a list of supported Linux distributions, see <a href="https://docs.microsoft.com/en-us/azure/virtual-machines/linux/endorsed-distros">Linux on Azure-Endorsed
Distributions</a>.</p>
</td>
</tr>
</tbody>
</table>
<br>
<h3 id="settings.gardener.cloud/v1.AzureSSHConfiguration">
<b>AzureSSHConfiguration</b>
</h3>
<p>
(<em>Appears on:</em>
<a href="#settings.gardener.cloud/v1.AzureLinuxConfiguration">AzureLinuxConfiguration</a>)
</p>
<p>
<p>AzureSSHConfiguration specifies the ssh key configuration for a Linux OS.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>publicKeys</code></br>
<em>
<a href="#settings.gardener.cloud/v1.AzureSSHPublicKey">
AzureSSHPublicKey
</a>
</em>
</td>
<td>
<p>The list of SSH public keys used to authenticate with linux based VMs.</p>
</td>
</tr>
</tbody>
</table>
<br>
<h3 id="settings.gardener.cloud/v1.AzureSSHPublicKey">
<b>AzureSSHPublicKey</b>
</h3>
<p>
(<em>Appears on:</em>
<a href="#settings.gardener.cloud/v1.AzureSSHConfiguration">AzureSSHConfiguration</a>)
</p>
<p>
<p>AzureSSHPublicKey the list of SSH public keys used to authenticate with linux based VMs.
key is placed.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>path</code></br>
<em>
string
</em>
</td>
<td>
<p>Specifies the full path on the created VM where ssh public key is stored. If the file already exists, the specified key is appended to the file. Example: /home/user/.ssh/authorized_keys</p>
</td>
</tr>
<tr>
<td>
<code>keyData</code></br>
<em>
string
</em>
</td>
<td>
<p>SSH public key certificate used to authenticate with the VM through ssh. The key needs to be at least 2048-bit and in ssh-rsa format.</p>
<p>For creating ssh keys, see [Create SSH keys on Linux and Mac for Linux VMs in Azure]<a href="https://docs.microsoft.com/azure/virtual-machines/linux/create-ssh-keys-detailed)">https://docs.microsoft.com/azure/virtual-machines/linux/create-ssh-keys-detailed)</a>.</p>
</td>
</tr>
</tbody>
</table>
<br>
<h3 id="settings.gardener.cloud/v1.AzureStorageProfile">
<b>AzureStorageProfile</b>
</h3>
<p>
(<em>Appears on:</em>
<a href="#settings.gardener.cloud/v1.AzureVirtualMachineProperties">AzureVirtualMachineProperties</a>)
</p>
<p>
<p>AzureStorageProfile is specifies the storage settings for the virtual machine disks.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>imageReference</code></br>
<em>
<a href="#settings.gardener.cloud/v1.AzureImageReference">
AzureImageReference
</a>
</em>
</td>
<td>
<p>Specifies information about the image to use. You can specify information about platform
images, marketplace images, or virtual machine images. This element is required when you want
to use a platform image, marketplace image, or virtual machine image, but is not used in other
creation operations.</p>
</td>
</tr>
<tr>
<td>
<code>osDisk</code></br>
<em>
<a href="#settings.gardener.cloud/v1.AzureOSDisk">
AzureOSDisk
</a>
</em>
</td>
<td>
<p>Specifies information about the operating system disk used by the virtual machine.
For more information about disks, see <a href="https://docs.microsoft.com/en-us/azure/virtual-machines/managed-disks-overview">About disks and VHDs for Azure virtual
machines</a>.</p>
</td>
</tr>
<tr>
<td>
<code>dataDisks</code></br>
<em>
<a href="#settings.gardener.cloud/v1.AzureDataDisk">
[]AzureDataDisk
</a>
</em>
</td>
<td>
<p>Specifies the parameters that are used to add a data disk to a virtual machine.
For more information about disks, see <a href="https://docs.microsoft.com/en-us/azure/virtual-machines/managed-disks-overview">About disks and VHDs for Azure virtual
machines</a>.</p>
</td>
</tr>
</tbody>
</table>
<br>
<h3 id="settings.gardener.cloud/v1.AzureSubResource">
<b>AzureSubResource</b>
</h3>
<p>
(<em>Appears on:</em>
<a href="#settings.gardener.cloud/v1.AzureVirtualMachineProperties">AzureVirtualMachineProperties</a>)
</p>
<p>
<p>AzureSubResource specifies information about the availability set that the virtual machine
should be assigned to. Virtual machines specified in the same availability set
are allocated to different nodes to maximize availability. For more information
about availability sets, see Manage the availability of virtual machines.</p>
<p>Currently, a VM can only be added to availability set at creation time.
The availability set to which the VM is being added should be under the
same resource group as the availability set resource. An existing VM cannot
be added to an availability set.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>id</code></br>
<em>
string
</em>
</td>
<td>
<p>This denotes the resource ID.</p>
</td>
</tr>
</tbody>
</table>
<br>
<h3 id="settings.gardener.cloud/v1.AzureSubnetInfo">
<b>AzureSubnetInfo</b>
</h3>
<p>
(<em>Appears on:</em>
<a href="#settings.gardener.cloud/v1.AzureProviderSpec">AzureProviderSpec</a>)
</p>
<p>
<p>AzureSubnetInfo is the information containing the subnet details</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>vnetName</code></br>
<em>
string
</em>
</td>
<td>
<p>The vNet Name.</p>
</td>
</tr>
<tr>
<td>
<code>vnetResourceGroup</code></br>
<em>
*string
</em>
</td>
<td>
<p>The resource group of the vNet.</p>
</td>
</tr>
<tr>
<td>
<code>subnetName</code></br>
<em>
string
</em>
</td>
<td>
<p>The name of the Subnet that will be utilised by the VM.</p>
</td>
</tr>
</tbody>
</table>
<br>
<h3 id="settings.gardener.cloud/v1.AzureVirtualMachineProperties">
<b>AzureVirtualMachineProperties</b>
</h3>
<p>
(<em>Appears on:</em>
<a href="#settings.gardener.cloud/v1.AzureProviderSpec">AzureProviderSpec</a>)
</p>
<p>
<p>AzureVirtualMachineProperties describes the properties of a Virtual Machine.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>hardwareProfile</code></br>
<em>
<a href="#settings.gardener.cloud/v1.AzureHardwareProfile">
AzureHardwareProfile
</a>
</em>
</td>
<td>
<p>Specifies the hardware settings for the virtual machine.</p>
</td>
</tr>
<tr>
<td>
<code>storageProfile</code></br>
<em>
<a href="#settings.gardener.cloud/v1.AzureStorageProfile">
AzureStorageProfile
</a>
</em>
</td>
<td>
<p>Specifies the storage settings for the virtual machine disks.</p>
</td>
</tr>
<tr>
<td>
<code>osProfile</code></br>
<em>
<a href="#settings.gardener.cloud/v1.AzureOSProfile">
AzureOSProfile
</a>
</em>
</td>
<td>
<p>Specifies the operating system settings used while creating the virtual
machine. Some of the settings cannot be changed once VM is provisioned.</p>
</td>
</tr>
<tr>
<td>
<code>networkProfile</code></br>
<em>
<a href="#settings.gardener.cloud/v1.AzureNetworkProfile">
AzureNetworkProfile
</a>
</em>
</td>
<td>
<p>Specifies the network interfaces of the virtual machine.</p>
</td>
</tr>
<tr>
<td>
<code>availabilitySet</code></br>
<em>
<a href="#settings.gardener.cloud/v1.AzureSubResource">
AzureSubResource
</a>
</em>
</td>
<td>
<p>Specifies information about the availability set that the virtual
machine should be assigned to. Virtual machines specified in the
same availability set are allocated to different nodes to maximize
availability. For more information about availability sets, see
Manage the availability of virtual machines.</p>
<p>Currently, a VM can only be added to availability set at creation
time. The availability set to which the VM is being added should
be under the same resource group as the availability set resource.
An existing VM cannot be added to an availability set.</p>
</td>
</tr>
<tr>
<td>
<code>identityID</code></br>
<em>
*string
</em>
</td>
<td>
<p>The identity of the virtual machine.</p>
</td>
</tr>
<tr>
<td>
<code>zone</code></br>
<em>
*int
</em>
</td>
<td>
<p>The virtual machine zone.</p>
</td>
</tr>
<tr>
<td>
<code>machineSet</code></br>
<em>
<a href="#settings.gardener.cloud/v1.AzureMachineSetConfig">
AzureMachineSetConfig
</a>
</em>
</td>
<td>
<p>AzureMachineSetConfig contains the information about the associated machineSet.</p>
</td>
</tr>
</tbody>
</table>
<hr/>
<p><em>
Generated with <a href="https://github.com/ahmetb/gen-crd-api-reference-docs">gen-crd-api-reference-docs</a>
</em></p>
