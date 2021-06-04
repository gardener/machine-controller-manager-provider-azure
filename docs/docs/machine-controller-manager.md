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

![Flowchart for working of Machine Controller Manager](../images/working-of-mcm.png)
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
| `providerSpec` | runtime.RawExtension | Provider Spec is the provider specific configuration to use during node creation. The schema for [`providerSpec`](#providerspec-schema) is defined here. |   
| `secretRef` | *corev1.SecretReference | SecretReference represents a Secret Reference. It has enough information to retrieve secret in any namespace. `SecretRef` stores the necessary secrets such as credentials or userdata. The schema for `secretRef` is defined here. More info: https://kubernetes.io/docs/concepts/configuration/secret |
| `credentialsSecretRef` | *corev1.SecretReference | SecretReference represents a Secret Reference. It has enough information to retrieve secret in any namespace. `CredentialsSecretRef` can optionally store the credentials (in this case the SecretRef does not need to store them). This might be useful if multiple machine classes with the same credentials but different user-datas are used. More info: https://kubernetes.io/docs/concepts/configuration/secret |

### FAQ

An FAQ is available [here](https://github.com/gardener/machine-controller-manager/blob/master/docs/FAQ.md)