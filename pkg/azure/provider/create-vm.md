# Create VM Flow (Current and Proposed)

## Current Flow:

Check if a vnetResourceGroup is configured.
- if yes then get the subnet resource. If there is any error getting the subnet return the error.
- Get NIC
    - If its not found then
        - create NIC
            - If that fails delete resources created till now and return error
    - If there is any other error delete resources created till now and return error
- Get image reference from the provider Spec.
    - If imageRef.URN is set then its a marketplace image.
        - Get vmImage
            - If there is any error delete resources created till now and return error
            - If image.Plan != nil
                - Get agreement
                    - if there is no agreement found or any other error delete resources created till now and return error
                    - If agreement is not accepted, then accept the agreement.
                        - If any error in update of agreement delete resources created till now and return error
- create VM parameters
- create the VM
- If there is any error then delete resources created till now and return error

VM creation pre-requisites
* Get subnet resource
* Get VMImage and handle agreement if its a marketplace image
* Check if NIC


## Revised flow:

### Pre-requisites

- Get Subnet
- Create armcompute.ImageReference from ProviderSpec
    - Check if image is a marketplace image. If yes then do the following
        - Get VM Image
            - If Plan exists then get the agreement. Keep he plan aside as it will be passed later when creating VM.
            - If the agreement exists but is not accepted, update the agreement after marking it as accepted.

> NOTE:
If any of the above fails then do not proceed further.
Till now no resources have been created and therefore if any of the above fails there is no need for a cleanup.

- Create NIC if it does not exist. Use the subnet fetched earlier.
    - If NIC creation fails then return error.
      NOTE: No clean up of resources required as VM and disks are still not been created.

- Create VM
    - Use ImageReference, Plan, NIC ID created above.
    - As part of VM creation OSDisk and any DataDisks will also get created.

  > NOTE: 
  > 
  > If the VM creation fails then there is really no need to delete the NIC.
  Discuss with the team on why is there a need. The VM creation will be attempted again,
  at which time if the NIC exists then creation of the NIC will be skipped.
The only case is when all attempts to create the VM has now exhausted and there
  will be no further attempt. At this point of time NIC created should be removed.
  If we can cleanly determine this then there will not be any need to delete the NIC
  on every failed attempt to create the VM. 
