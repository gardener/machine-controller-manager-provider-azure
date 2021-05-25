# Machine Controller Manager

## Overview
Machine Controller Manager aka MCM is a group of cooperative controllers that manage the lifecycle of the worker machines. It is inspired by the design of Kube Controller Manager in which various sub controllers manage their respective Kubernetes Clients. MCM gives you the following benefits:

- seamlessly manage machines/nodes with a declarative API (of course, across different cloud providers)
- integrate generically with the cluster autoscaler
- plugin with tools such as the node-problem-detector
- transport the immutability design principle to machine/nodes
- implement e.g. rolling upgrades of machines/nodes

## Design of Machine Controller Manager
The design of the Machine Controller Manager is influenced by the Kube Controller Manager, where-in multiple sub-controllers are used to manage the Kubernetes clients.

### Design Principles

It's designed to run in the master plane of a Kubernetes cluster. It follows the best principles and practices of writing controllers, including, but not limited to:

- Reusing code from kube-controller-manager
- Leader election to allow HA deployments of the controller
- `workqueues` and multiple thread-workers
- `SharedInformers` that limit to minimum network calls, de-serialization and provide helpful create/update/delete events for resources
- Rate-limiting to allow back-off in case of network outages and general instability of other cluster components
- Sending events to respected resources for easy debugging and overview
- Prometheus metrics, health and (optional) profiling endpoints

### Objects of Machine Controller Manager

Machine Controller Manager reconciles a set of Custom Resources namely `MachineDeployment`, `MachineSet` and `Machines` which are managed & monitored by their controllers MachineDeployment Controller, MachineSet Controller, Machine Controller respectively along with another cooperative controller called the Safety Controller.

Machine Controller Manager makes use of 4 CRD objects and 1 Kubernetes secret object to manage machines. They are as follows:
| Custom ResourceObject | Description |
| --- | --- |
| `MachineClass`| A `MachineClass` represents a template that contains cloud provider specific details used to create machines.|
| `Machine`| A `Machine` represents a VM which is backed by the cloud provider.|
| `MachineSet` | A `MachineSet` ensures that the specified number of `Machine` replicas are running at a given point of time.|
| `MachineDeployment`| A `MachineDeployment` provides a declarative update for `MachineSet` and `Machines`.|
| `Secret`| A `Secret` here is a Kubernetes secret that stores cloudconfig (initialization scripts used to create VMs) and cloud specific credentials.|

### Associated Controllers of Machine Controller Manager

<table>
    <thead>
        <tr>
            <th>Controller</th>
            <th>Description</th>
        </tr>
    </thead>
    <tbody>
        <tr>
            <td>MachineDeployment controller</td>
            <td>Machine Deployment controller reconciles the <code>MachineDeployment</code> objects and manages the lifecycle of <code>MachineSet</code> objects. <code>MachineDeployment</code> consumes provider specific <code>MachineClass` in its <code>spec.template.spec</code> which is the template of the VM spec that would be spawned on the cloud by MCM.</td>
        </tr>
        <tr>
            <td>MachineSet controller</td>
            <td>MachineSet controller reconciles the <code>MachineSet</code> objects and manages the lifecycle of <code>Machine</code> objects.</td>
        </tr>
        <tr>
            <td>Safety controller</td>
            <td>There is a Safety Controller responsible for handling the unidentified or unknown behaviours from the cloud providers. Safety Controller:
                <ul>
                    <li>
                        freezes the MachineDeployment controller and MachineSet controller if the number of <code>Machine</code> objects goes beyond a certain threshold on top of <code>Spec.replicas</code>. It can be configured by the flag <code>--safety-up</code> or <code>--safety-down</code> and also <code>--machine-safety-overshooting-period`</code>.
                    </li>
                    <li>
                        freezes the functionality of the MCM if either of the <code>target-apiserver</code> or the <code>control-apiserver</code> is not reachable.
                    </li>
                    <li>
                        unfreezes the MCM automatically once situation is resolved to normal. A <code>freeze</code> label is applied on <code>MachineDeployment</code>/<code>MachineSet</code> to enforce the freeze condition.
                    </li>
                </ul>
            </td>
        </tr>
    </tbody>
</table>

Along with the above Custom Controllers and Resources, MCM requires the `MachineClass` to use K8s `Secret` that stores cloudconfig (initialization scripts used to create VMs) and cloud specific credentials. All these controllers work in an co-operative manner. They form a parent-child relationship with `MachineDeployment` Controller being the grandparent, `MachineSet` Controller being the parent, and `Machine` Controller being the child.


## Working of MCM

![Flowchart for working of Machine Controller Manager](./images/working-of-mcm.png)
*Figure 1: Flowchart for working of Machine Controller Manager*

In MCM, there are two K8s clusters in the scope — a Control Cluster and a Target Cluster. Control Cluster is the K8s cluster where the MCM is installed to manage the machine lifecycle of the Target Cluster. In other words, Control Cluster is the one where the `machine-*` objects are stored. Target Cluster is where all the `node` objects are registered. These clusters can be two distinct clusters or the same cluster, whichever fits.

When a `MachineDeployment` object is created, MachineDeployment Controller creates the corresponding `MachineSet` object. The MachineSet Controller in-turn creates the `Machine` objects. The Machine Controller then talks to the cloud provider API and actually creates the VMs on the cloud.

The cloud initialization script that is introduced into the VMs via the K8s `Secret` consumed by the `MachineClasses` talks to the KCM (K8s Controller Manager) and creates the `node` objects. `Nodes` after registering themselves to the Target Cluster, start sending health signals to the `Machine` objects. That is when MCM updates the status of the `Machine` object from `Pending` to `Running`. 

## Specification
### Schema

In the following description, a field that is italicised can be considered optional.

#### **MachineClass**

| Field Name | Type | Description |
| --- | --- | --- |
| `apiVersion` | string | APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources|
| `kind` | string | Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |
| `metadata` | metav1.ObjectMeta | Standard object's metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata |
| `provider` | string | Provider is the combination of name and location of cloud specific drivers |
| `providerSpec` | runtime.RawExtension | Provider Spec is the provider specific configuration to use during node creation. The schema for `providerSpec` is defined here. |   
| `secretRef` | *corev1.SecretReference | SecretReference represents a Secret Reference. It has enough information to retrieve secret in any namespace. `SecretRef` stores the necessary secrets such as credentials or userdata. The schema for `secretRef` is defined here. More info: https://kubernetes.io/docs/concepts/configuration/secret |
| `credentialsSecretRef` | *corev1.SecretReference | SecretReference represents a Secret Reference. It has enough information to retrieve secret in any namespace. `CredentialsSecretRef` can optionally store the credentials (in this case the SecretRef does not need to store them). This might be useful if multiple machine classes with the same credentials but different user-datas are used. More info: https://kubernetes.io/docs/concepts/configuration/secret |

### FAQ

An FAQ is available [here](https://github.com/gardener/machine-controller-manager/blob/master/docs/FAQ.md)

# Machine Controller Manager Provider Azure

MCM supports declarative management of machines in a K8s Cluster on various cloud providers like AWS, Azure, GCP, AliCloud, OpenStack, Metal-stack, Packet, KubeVirt, VMWare, Yandex. It can, of course, be easily extended to support other cloud providers.

Going ahead having the implementation of the Machine Controller Manager supporting too many cloud providers would be too much upkeep from both a development and a maintenance point of view. Which is why, the Machine Controller component of MCM has been moved to Out-of-Tree design where Machine Controller for respective cloud provider runs as an independent executable; even though typically packaged under the same deployment.

This [Machine Controller Manager Provider Azure](https://github.com/gardener/machine-controller-manager-provider-azure) will implement a common interface to manage the VMs on the Azure cloud provider. Now, while Machine Controller deals with the `Machine` objects, Machine Controller Manager (MCM) deals with higher level objects such as `MachineSet` and `MachineDeployment` objects.

## Fundamental Design Principles
Following are the basic development principles for this external plugin:
* Communication between this Machine Controller (MC) and Machine Controller Manager (MCM) is achieved using the Kubernetes native declarative approach.
* Machine Controller (MC) behaves as the controller used to interact with the cloud provider Azure and manages the VMs corresponding to the `Machine` objects.
* Machine Controller Manager (MCM) deals with higher level objects such as `MachineSet` and `MachineDeployment` objects.


## Associated controllers of Machine Controller Manager Provider Azure
<table>
    <thead>
        <tr>
            <th>Controller</th>
            <th>Description</th>
        </tr>
    </thead>
    <tbody>
        <tr>
            <td>Machine controller</td>
            <td>
                Machine controller reconciles the machine objects and creates the actual instances of the machines on the cloud by communicating with the cloud provider APIs.
            </td>
        </tr>
        <tr>
            <td>Safety controller</td>
            <td>
                Safety controller in the machine controller manages the orphan VMs.
                <ul>
                    <li>
                        It lists all the VMs in the cloud; matching the tag of given cluster name and maps the VMs with the <code>Machine`</code> objects using the <code>ProviderID</code> field. VMs without any backing <code>Machine</code> objects are logged and deleted after confirmation.
                    </li>
                    <li>
                        This handler runs every 30 minutes and is configurable via <code>--machine-safety-orphan-vms-period</code> flag.
                    </li>
                </ul>
            </td>
        </tr>
    </tbody>
</table>

## Specification 
### Schema
#### **MachineClass.ProviderSpec**

Provider Spec is the provider specific configuration to use during node creation. The provider specification for Azure is defined as below:

| FieldName | Type | Description |
| --- | --- | --- |
| `location` | string | The resource location.|
| `resourceGroup` | string | The name of the resource group.|
| `subnetInfo` | [AzureSubnetInfo](#AzureSubnetInfo) | `subnetInfo` holds the information about the subnet resource.|
| `properties` | [AzureVirtualMachineProperties](#AzureVirtualMachineProperties) | `properties` describes the properties of a Virtual Machine.|
| `tags` | map[string]string | `tags` are set of key value pairs labelled on virtual machine to organise them into a taxonomy. |

</br>

#### **AzureSubnetInfo**
`AzureSubnetInfo` holds the information about the subnet resource.

| FieldName | Type | Description |
| --- | --- | --- |
| `subnetName` | string | Name of the subnet resource.|
| `vnetName`| string | Name of the vnet resource. |
| `vnetResourceGroup` | string | Name of the resource group to which vnet belongs to. |

</br>

#### **AzureVirtualMachineProperties**
`AzureVirtualMachineProperties` describes the properties of a Virtual Machine.

| FieldName | Type | Description |
| --- | --- | --- |
| `identityID` | string | The identity of the virtual machine |
| `zone` | integer | The virtual machine zone.|
| `availabilitySet` | [AzureSubResource](#AzureSubResource) | Specifies information about the availability set that the virtual machine should be assigned to. Virtual machines specified in the same availability set are allocated to different nodes to maximize availability. For more information about availability sets, see Manage the availability of virtual machines. </br></br> Currently, a VM can only be added to availability set at creation time. The availability set to which the VM is being added should be under the same resource group as the availability set resource. An existing VM cannot be added to an availability set. |
| `hardwareProfile` | [AzureHardwareProfile](#AzureHardwareProfile) | Specifies the hardware settings for the virtual machine.|
| `machineSet` | [AzureMachineSetConfig](#AzureMachineSetConfig) | AzureMachineSetConfig contains the information about the associated `machineSet`.|
| `networkProfile` | [AzureNetworkProfile](#AzureNetworkProfile) | Specifies the network interfaces of the virtual machine. | 
| `osProfile` | [AzureOSProfile](#AzureOSProfile) | Specifies the operating system settings used while creating the virtual machine. Some of the settings cannot be changed once VM is provisioned.|
| `storageProfile` | AzureStorageProfile | Specifies the storage settings for the virtual machine disks.|

</br>

#### **AzureSubResource**
Specifies information about the availability set that the virtual machine should be assigned to. Virtual machines specified in the same availability set are allocated to different nodes to maximize availability. For more information about availability sets, see Manage the availability of virtual machines.

Currently, a VM can only be added to availability set at creation time. The availability set to which the VM is being added should be under the same resource group as the availability set resource. An existing VM cannot be added to an availability set.

|FieldName|Type|Description|
| --- | --- | --- |
| `id` | string | This denotes the resource ID.|

</br>

#### **AzureHardwareProfile**
Specifies the hardware settings for the virtual machine.


<table>
    <thead>
        <tr>
            <th>FieldName</th>
            <th>Type</th>
            <th>Description</th>
        </tr>
    </thead>
    <tbody>
        <tr>
            <td>
                vmSize
            </td>
            <td>
                string
            </td>
            <td>
                Specifies the size of the virtual machine. The enum data type is currently deprecated and will be removed by December 23rd 2023. Recommended way to get the list of available sizes is using these APIs:
                <ul>
                    <li>
                        <a href="https://docs.microsoft.com/en-us/rest/api/compute/availabilitysets/listavailablesizes">List all available virtual machine sizes in an availability set </a>
                    </li>
                    <li>
                        <a href="https://docs.microsoft.com/en-us/rest/api/compute/resourceskus/list">List all available virtual machine sizes in a region </a>
                    </li>
                    <li>
                        <a href="https://docs.microsoft.com/en-us/rest/api/compute/virtualmachines/listavailablesizes">List all available virtual machine sizes for resizing.</a>
                    </li>
                </ul>
                The available VM sizes depend on region and availability set.
            </td>
        </tr>
    </tbody>
</table>

</br>

#### **AzureMachineSetConfig**
AzureMachineSetConfig contains the information about the associated machineSet.

| FieldName | Type | Description |
| --- | --- | --- |
| id | string | |
| kind | string | |

</br>

#### **AzureNetworkProfile**
Specifies the network interfaces of the virtual machine.

| FieldName | Type | Description |
| --- | --- | --- |
| `networkInterfaces` | [AzureNetworkInterfaceReference](#AzureNetworkInterfaceReference) | Specifies the network interfaces of the virtual machine. |
| `acceleratedNetworking` | boolean | Specifies if the acceleration is enabled in network  |

</br>

#### **AzureNetworkInterfaceReference** 
Specifies the network interfaces of the virtual machine. 

| FieldName | Type | Description |
| --- | --- | --- |
| `id` | string | Resource Id |
| `properties` | [AzureNetworkInterfaceReferenceProperties](#AzureNetworkInterfaceReferenceProperties) | Specifies the primary network interface in case the virtual machine has more than 1 network interface.|

</br>

#### **AzureNetworkInterfaceReferenceProperties**
| FieldName | Type | Description |
| --- | --- | --- |
| `primary` | boolean | Specifies the primary network interface in case the virtual machine has more than 1 network interface. |

</br>

#### **AzureOSProfile**
Specifies the operating system settings for the virtual machine. Some of the settings cannot be changed once VM is provisioned. For more details see [documentation on osProfile in Azure Virtual Machines](https://docs.microsoft.com/en-us/rest/api/compute/virtualmachines/createorupdate#osprofile)

| FieldName | Type | Description |
| --- | --- | --- |
| `computerName` | string |Specifies the host OS name of the virtual machine. This name cannot be updated after the VM is created. |
|`adminUsername` |string|Specifies the name of the administrator account. This property cannot be updated after the VM is created.|
|`adminPassword`|string|Specifies the password of the administrator account. |
|`customData`|string|Specifies a base-64 encoded string of custom data. The base-64 encoded string is decoded to a binary array that is saved as a file on the Virtual Machine. The maximum length of the binary array is 65535 bytes. </br> </br> <b>Note: Do not pass any secrets or passwords in customData property</b> |
|`linuxConfiguration`|[AzureLinuxConfiguration](#AzureLinuxConfiguration)|Specifies the Linux operating system settings on the virtual machine. For a list of supported Linux distributions, see [Linux on Azure-Endorsed Distributions](https://docs.microsoft.com/en-us/azure/virtual-machines/virtual-machines-linux-endorsed-distros?toc=/azure/virtual-machines/linux/toc.json) |

</br>

#### **AzureLinuxConfiguration**
Specifies the Linux operating system settings on the virtual machine. 

| FieldName | Type | Description |
| --- | --- | --- |
| `disablePasswordAuthentication` | boolean |Specifies whether password authentication should be disabled. |
|`ssh`|[AzureSSHConfiguration](#AzureSSHConfiguration)|Specifies the ssh key configuration for a Linux OS.|

</br>

#### **AzureSSHConfiguration**

| FieldName | Type | Description |
| --- | --- | --- |
| `publicKeys` | [AzureSSHPublicKey](#AzureSSHPublicKey) | Specifies the ssh key for a Linux OS.|

</br>

#### **AzureSSHPublicKey**
Specifies the ssh key configuration for a Linux OS.

| FieldName | Type | Description |
| --- | --- | --- |
| `path` | string |Specifies the full path on the created VM where ssh public key is stored. If the file already exists, the specified key is appended to the file. Example: `/home/user/.ssh/authorized_keys` |
| `keyData` | string |SSH public key certificate used to authenticate with the VM through ssh. The key needs to be at least 2048-bit and in ssh-rsa format.  |

</br>

## Usage of the Azure OOT

### Running Machine Controller for Azure

1. Open a terminal pointing to `$GOPATH/src/github.com/gardener/` folder. Clone [Machine Controller Manager Provider Azure](https://github.com/gardener/machine-controller-manager-provider-azure) using
    ```bash
    git clone https://github.com/gardener/machine-controller-manager-provider-azure.git
    ```

2. Navigate to `machine-controller-manager-provider-azure` folder
    ```bash
    cd machine-controller-manager-provider-azure
    ```
    1. In the `MAKEFILE` make sure that 
        - `$TARGET_KUBECONFIG` points to the kubeconfig file of the Target Cluster (whose machines are to be managed). This points to the `shoot` in the context of Gardener.
        - `$CONTROL_KUBECONFIG` points to the kubeconfig file of the Control Cluster (which holds the machine CRs of the Target Cluster). This points to the `seed` in the context of Gardener.
        - `$CONTROL_NAMESPACE` represents the namespaces where MCM is looking for `Machine` objects. 

    2. Run the machine controller (driver) using the command below.
        ```bash
        make start
        ```

### Running Machine Controller Manager
1. Open another terminal pointing to `$GOPATH/src/github.com/gardener`,
    1. Clone the [latest MCM](https://github.com/gardener/machine-controller-manager) using
    ```bash
    git clone https://github.com/gardener/machine-controller-manager.git
    ```
    1. Navigate to the newly created directory.
    ```bash
    cd machine-controller-manager
    ```
    1. Make same changes/entries as done in the point 2 of section [Running Machine Controller for Azure](#Running-Machine-Controller-for-Azure)
    1. Deploy the required CRDs from the machine-controller-manager repo,
    ```bash
    kubectl apply -f kubernetes/crds.yaml
    ```
    1. Run the machine-controller-manager
    ```bash
    make start
    ```

### Managing the machines with declarative updates
1. On the third terminal pointing to `$GOPATH/src/github.com/gardener/machine-controller-manager-provider-azure`, fill in the YAML files for `secret`, `MachineClass` and `Machine` from the `./kubernetes` folder with appropriate values and deploy them.

    ```bash
    kubectl apply -f kubernetes/secret.yaml
    kubectl apply -f kubernetes/machine-class.yaml
    kubectl apply -f kubernetes/machine.yaml
    ```

    Once machine joins, you can test by deploying a `MachineDeployment` object.

3. Deploy the `MachineDeployment` object and make sure it joins the cluster successfully.
    ```bash
    kubectl apply -f kubernetes/machine-deployment.yaml
    ```
4. Make sure to delete both the `Machine` and `MachineDeployment` object after use.
    ```bash
    kubectl delete -f kubernetes/machine.yaml
    kubectl delete -f kubernetes/machine-deployment.yaml
    ```
