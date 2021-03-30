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
- leader election to allow HA deployments of the controller
- `workqueues` and multiple thread-workers
- `SharedInformers` that limit to minimum network calls, de-serialization and provide helpful create/update/delete events for resources
- rate-limiting to allow back-off in case of network outages and general instability of other cluster components
- sending events to respected resources for easy debugging and overview
- Prometheus metrics, health and (optional) profiling endpoints

### Objects of Machine Controller Manager

Machine Controller Manager reconciles a set of Custom Resources namely `MachineDeployment`, `MachineSet` and `Machines` which are managed & monitored by their controllers MachineDeployment Controller, MachineSet Controller, Machine Controller respectively along with another cooperative controller called the Safety Controller.

Machine Controller Manager makes use of 4 CRD objects and 1 Kubernetes secret object to manage machines. They are as follows:
1. `MachineClass`: Represents a template that contains cloud provider specific details used to create machines.
1. `Machine`: Represents a VM which is backed by the cloud provider.
1. `MachineSet`: Represents a group of machines managed by the Machine Controller Manager.
1. `MachineDeployment`: Represents a group of machine-sets managed by the Machine Controller Manager to allow updating machines.
1. `Secret`: Represents a Kubernetes secret that stores cloudconfig (initialization scripts used to create VMs) and cloud specific credentials

### Components of Machine Controller Manager

- `MachineDeployment` provides a declarative update for `MachineSet` and `Machines`. MachineDeployment Controller reconciles the `MachineDeployment` objects and manages the lifecycle of `MachineSet` objects. `MachineDeployment` consumes provider specific `MachineClass` in its `spec.template.spec` which is the template of the VM spec that would be spawned on the cloud by MCM.
- `MachineSet` ensures that the specified number of `Machine` replicas are running at a given point of time. MachineSet Controller reconciles the `MachineSet` objects and manages the lifecycle of `Machine` objects.
- `Machines` are the backing objects for the actual VMs running on one of the supported cloud platforms. Machine Controller is the controller that actually communicates with the cloud provider to create/update/delete machines on the cloud.
- There is a Safety Controller responsible for handling the unidentified or unknown behaviours from the cloud providers.

Along with the above Custom Controllers and Resources, MCM requires the `MachineClass` to use K8s `Secret` that stores cloudconfig (initialization scripts used to create VMs) and cloud specific credentials. All these controllers work in an co-operative manner. They form a parent-child relationship with `MachineDeployment` Controller being the grandparent, `MachineSet` Controller being the parent, and `Machine` Controller being the child.

### More on Safety Controller

Safety Controller undertakes the following responsibilities:

**Orphan VM handling:**

- It lists all the VMs in the cloud; matching the tag of given cluster name and maps the VMs with the `Machine` objects using the `ProviderID` field. VMs without any backing `Machine` objects are logged and deleted after confirmation.
- This handler runs every 30 minutes and is configurable via `--machine-safety-orphan-vms-period` flag.

**Freeze mechanism:**

- Safety Controller freezes the MachineDeployment and MachineSet controller if the number of `Machine` objects goes beyond a certain threshold on top of `Spec.replicas`. It can be configured by the flag `--safety-up` or `--safety-down` and also `--machine-safety-overshooting-period`.
- Safety Controller freezes the functionality of the MCM if either of the target-apiserver or the control-apiserver is not reachable.
- Safety Controller unfreezes the MCM automatically once situation is resolved to normal. A freeze label is applied on `MachineDeployment`/`MachineSet` to enforce the freeze condition.

### Working of MCM

![Flowchart for working of Machine Controller Manager](./images/working-of-mcm.png)
*Figure 1: Flowchart for working of Machine Controller Manager*

In MCM, there are two K8s clusters in the scope — a Control Cluster and a Target Cluster. Control Cluster is the K8s cluster where the MCM is installed to manage the machine lifecycle of the Target Cluster. In other words, Control Cluster is the one where the `machine-*` objects are stored. Target Cluster is where all the `node` objects are registered. These clusters can be two distinct clusters or the same cluster, whichever fits.

When a `MachineDeployment` object is created, MachineDeployment Controller creates the corresponding `MachineSet` object. The MachineSet Controller in-turn creates the `Machine` objects. The Machine Controller then talks to the cloud provider API and actually creates the VMs on the cloud.

The cloud initialization script that is introduced into the VMs via the K8s `Secret` consumed by the `MachineClasses` talks to the KCM (K8s Controller Manager) and creates the `node` objects. `Nodes` after registering themselves to the Target Cluster, start sending health signals to the `Machine` objects. That is when MCM updates the status of the `Machine` object from `Pending` to `Running`. 

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
