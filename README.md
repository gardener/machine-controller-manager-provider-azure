# machine-controller-manager-provider-azure

[![REUSE status](https://api.reuse.software/badge/github.com/gardener/machine-controller-manager-provider-azure)](https://api.reuse.software/info/github.com/gardener/machine-controller-manager-provider-azure)

Out of tree (controller based) implementation for `Azure` as a new provider.

## About
The Azure Out Of Tree provider implements the interface defined at [MCM OOT driver](https://github.com/gardener/machine-controller-manager/blob/master/pkg/util/provider/driver/driver.go).

## Fundamental Design Principles:
Following are the basic development principles for this external plugin:
* Communication between this Machine Controller (MC) and Machine Controller Manager (MCM) is achieved using the Kubernetes native declarative approach.
* Machine Controller (MC) behaves as the controller used to interact with the cloud provider Microsoft Azure and manage the VMs corresponding to the machine objects.
* Machine Controller Manager (MCM) deals with higher level objects such as `MachineSet` and `MachineDeployment` objects.

## Usage of the Azure OOT

1. Open terminal pointing to `$GOPATH/src/github.com/gardener`. Clone this repository. 

2. Navigate to `$GOPATH/src/github.com/gardener/machine-controller-manager-provider-azure`. 
    - In the `MAKEFILE` make sure that 
      - `$TARGET_KUBECONFIG` points to the kubeconfig file of the cluster where you wish to manage machines. This points to the shoot cluster in the context of Gardener.
      - `$CONTROL_KUBECONFIG` points to the kubeconfig file of the cluster which holds these machine CRs. This points to the seed cluster in the context of Gardener.
      - `$CONTROL_NAMESPACE` represents the namespaces where MCM is looking for machine CR objects. 

    - Run the machine controller (driver) using the command below.
        ```bash
        make start
        ```
3. On the second terminal pointing to `$GOPATH/src/github.com/gardener`,
    - Clone the [latest MCM code](https://github.com/gardener/machine-controller-manager)
        ```bash
        git clone git@github.com:gardener/machine-controller-manager.git
        ```
    - Navigate to the newly created directory.
        ```bash
        cd machine-controller-manager
        ```
    - Deploy the required CRDs from the machine-controller-manager repo,
        ```bash
        kubectl apply -f kubernetes/crds.yaml
        ```
    - Run the machine-controller-manager
        ```bash
        make start
        ```
4. On the third terminal pointing to `$GOPATH/src/github.com/gardener/machine-controller-manager-provider-azure`

    Fill in the object files given below and deploy them as described below:

    - Deploy `kubernetes secret` for Azure.
        ```bash
        kubectl apply -f kubernetes/secret.yaml
        ```
    - Deploy `MachineClass`
        ```bash
        kubectl apply -f kubernetes/machine-class.yaml
        ```
    
    - Deploy `Machine` object and make sure it joins the cluster successfully.
        ```bash
        kubectl apply -f kubernetes/machine.yaml
        ```
    - Once machine joins, you can test by deploying a machine-deployment.

    - Deploy the `MachineDeployment` object and make sure it joins the cluster successfully.
        ```bash
        kubectl apply -f kubernetes/machine-deployment.yaml
        ```
    - Make sure to delete both the `Machine` and `MachineDeployment` object after use.
        ```bash
        kubectl delete -f kubernetes/machine.yaml
        kubectl delete -f kubernetes/machine-deployment.yaml
        ```

## Support for a new provider
- Steps to be followed while implementing/testing a new provider are mentioned [here](https://github.com/gardener/machine-controller-manager/blob/master/docs/development/cp_support_new.md)
