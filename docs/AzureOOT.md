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
