
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


