# OSSM Federation Failover Demo

Sample scripts for setting a bookinfo ratings service failover demo.

## Versions

| Name         | Version       |
| --           | --            |
| OCP AWS      | 4.9.29        |
| OSSM         | 2.1.2         |

## Prerequisite

- OCP AWS cluster01 (west-mesh) has been provisioned in region `us-west-2`
- OCP AWS cluster02 (east-mesh) has been provisioned in region `us-east-2`

- OSSM operators 2.1.2 have been installed on both clusters
- oc client can login both clusters using their kubeconfig file  

## How to Run

1. Setup
```
$ export MESH1_KUBECONFIG=<cluster01 kubeconfig file path>
$ export MESH2_KUBECONFIG=<cluster02 kubeconfig file path>
$ source common.sh 
$ oc1 login -u <cluster01 admin user> -p <password> --server=<cluster01 api server> --insecure-skip-tls-verify=true
$ oc2 login -u <cluster02 admin user> -p <password> --server=<cluster02 api server> --insecure-skip-tls-verify=true

$ ./setup.sh
$ ./install.sh
# If an AWS LB creation is slow when too many clusters running in the same region,
# and if servicemeshpeer connection is false. Then 
# Wait 10 minutes and then run install.sh again.

# Depends on the network condition from a cloud provider, you may adjust the 
# destinationrule-failover.yaml .spec.trafficPolicy.outlierDetection.

```

2. Import Ratings Service and Enable failover
```
$ oc1 apply -f export/exportedserviceset.yaml
$ oc2 apply -f import/importedserviceset.yaml
$ oc2 -n east-mesh-system get importedservicesets west-mesh -o json
# Wait 10 minutes for pilot pushing all updates
$ oc2 apply -f examples/destinationrule-failover.yaml
```

By default, LoadBalancer uses ROUND_ROBIN algorithm and it spreads traffic across local and remote endpoints.

The example destinationrule above uses LEAST_CONN algorithm for routing traffic to a local endpoint first.
LEAST_CONN is deprecated from the latest upstream Istio. A future OSSM release will replace LEAST_CONN with LEAST_REQUEST.

Reference: https://istio.io/latest/docs/reference/config/networking/destination-rule/#LoadBalancerSettings-SimpleLB

## How to Verify

- Refresh cluster02 (east-mesh) boookinfo productpage.

  e.g. `curl  http://<bookinfo route from east-mesh-system>.apps.<cluster02 aws hostname>/productpage`

  And check cluster02 bookinfo-ha ns pod ratings-v1 ratings container log
  for example,
  ```
  Server listening on: http://0.0.0.0:9080
    GET /ratings/0
  ```

  Check cluster01 bookinfo-ha ns pod ratings-v1 ratings container log.
  There is no new GET request coming in cluster01 after applying the DestinationRule.

- Check kiali graph from east-mesh

- Scale ratings-v1 deployment in cluster02 bookinfo-ha to 0

- Refresh cluster02 (east-mesh) boookinfo productpage
- Check cluster01 bookinfo-ha ns pod ratings-v1 ratings container log.
  ```
  Server listening on: http://0.0.0.0:9080
    GET /ratings/0
  ```

- Check kiali graph from east-mesh

- Restore ratings-v1 deployment in cluster02 bookinfo-ha to 1

- Refresh cluster02 (east-mesh) boookinfo productpage
- Check cluster02 bookinfo-ha ns pod ratings-v1 ratings container log.
  ```
  Server listening on: http://0.0.0.0:9080
    GET /ratings/0
  ```
