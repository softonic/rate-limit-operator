[![Go Report Card](https://goreportcard.com/badge/softonic/rate-limit-operator)](https://goreportcard.com/report/softonic/rate-limit-operator)
[![Releases](https://img.shields.io/github/release-pre/softonic/rate-limit-operator.svg?sort=semver)](https://github.com/softonic/rate-limit-operator/releases)
[![LICENSE](https://img.shields.io/github/license/softonic/rate-limit-operator.svg)](https://github.com/softonic/rate-limit-operator/blob/master/LICENSE)
[![DockerHub](https://img.shields.io/docker/pulls/softonic/rate-limit-operator.svg)](https://hub.docker.com/r/softonic/rate-limit-operator)


# rate-limit-operator
Rate Limit operator for Envoy Proxy

# Quick Start

Steps to got Operator working

- Deploy the Operator ( kubectl or helm )
- Deploy RateLimit CR

## Deployment

### Requirements

In this example we assume you already have a k8s cluster running

### Deploy using kubectl 

```bash
$ make deploy
```

You can find public image in the softonic/rate-limit-operator docker hub repository.

### Deploy using Helm

```bash
$ make helm-deploy
```


## Create a CR ratelimit

```bash
$ kubectl apply -f config/samples/networking_v1alpha1_ratelimit.yaml
```

# DEVEL ENVIRONMENT

Compile the code and deploy the needed resources

```bash
$ make generate
$ make run
```


# Motivation


The operator will help you configure the necessary resources to get to this

https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/rate_limit_filter#config-http-filters-rate-limit

from a CR that you will need to provide.

This operator will need this CR, and from this configuration will create the necessary envoyfilters resources in istio
control plane, and the necessary configs ratelimit system 

https://github.com/envoyproxy/ratelimit

needs. 

This is accomplished with a new CRD.

In this new CRD, you can set different configuration depending on your needs.

Here is an example you can start with

```
apiVersion: networking.softonic.io/v1alpha1
kind: RateLimit
metadata:
  name: ratelimit-sample
  finalizers:
  - ratelimit.networking.softonic.io
spec:
  targetRef:
    apiVersion: "networking.istio.io/v1alpha3"
    kind:       VirtualService
    name:       vs-test
    namespace:  ratelimitoperatortest
  destinationCluster: "outbound|80||chicken-head-nginx.chicken-head.svc.cluster.local"
  unit: seconds
  requestPerUnit: 100
  dimensions:
  - request_header:
      descriptor_key: remote_address
      header_name: x-custom-ip
  workloadselector:
    app: istio-ingressgateway
```

* targetRef will point to a VS that will have the host field that you will need in your envoyfilter in order to apply the routing
* destination cluster is the Cluster ( here cluster is referring to the concept of cluster in the envoy/istio language ) 
* following configuration refer to what to limit. In this case we are limiting 100 per second per x-custom-ip
* workloadselector mean that will set the envoyfilters in the ingressgateway ( you need to have these labels in the ingressgateway deployment )


# Diagram


![Image of operator flow](/docs/diagram.png)
