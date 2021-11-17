# OctoMuffins

This is a demo of SBOM generation, container signing, and signature verification.

## Dependencies

The basic dependencies are 

- Kubernetes Cluster
- Container Registry
- cosign
- ko

But in this demo I am also leveraging [Knative Serving](https://knative.dev) and a simple CloudEvents viewer called [Sockeye](https://github.com/n3wscott/sockeye). 


## Setup

> _NOTE_: This assumed kind 0.11.1

```shell
cat <<EOF | kind create cluster --name octomuffin --wait 120s --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  image: kindest/node:v1.20.7@sha256:cbeaf907fc78ac97ce7b625e4bf0de16e3ea725daf6b04f930bd14c67c671ff9
  extraPortMappings:
  - containerPort: 31080
    listenAddress: 127.0.0.1
    hostPort: 80
EOF
```

Then after the kind cluster is up,

```shell
curl -sL https://raw.githubusercontent.com/csantanapr/knative-kind/master/02-serving.sh | sh
curl -sL https://raw.githubusercontent.com/csantanapr/knative-kind/master/02-contour.sh | sh
kubectl patch configmap -n knative-serving config-domain -p "{\"data\": {\"127.0.0.1.nip.io\": \"\"}}"
```

All this to get Sockeye:

```shell
kubectl apply -f https://github.com/n3wscott/sockeye/releases/download/v0.7.0/release.yaml
```

I am going to use ghcr.io for my container registry, we need to export a `KO_DOCKER_REPO`,

> _NOTE_: You can use any registry you would like.

> _NOTE_: I had to make each image public to let it run in kind.

```shell
export KO_DOCKER_REPO=ghcr.io/n3wscott/octomuffin
```

## Demo

> _NOTE_: The following demo is assumed to be in the root of the octomuffins project. 

To start we need to build and capture our container image digest,

```shell
OCTO_IMAGE=`ko publish --platform=all .`
```

### SBOM

Going to use [spdx-sbom-generator](https://github.com/opensbom-generator/spdx-sbom-generator).

Running `spdx-sbom-generator` produces `./bom-go-mod.spdx`. 

To attach this to the container, 

```shell
cosign attach sbom $OCTO_IMAGE
```

> _HINT_: To confirm the SBOM was uploaded, 
> ```shell
>   crane ls `echo $OCTO_IMAGE | cut -f1 -d"@"`
> ```

### Container Signing

```shell
COSIGN_EXPERIMENTAL=1 cosign sign $OCTO_IMAGE 
```

### Container Validation

```shell
COSIGN_EXPERIMENTAL=1 cosign verify $OCTO_IMAGE
```

### Bonus Cosigned

Download and install cosign, 

```shell
git checkout https://github.com/sigstore/cosign.git
cd cosign
ko apply -Bf config/ --platform=all
```

Create a restricted namespace,

```shell
kubectl create namespace restricted
kubectl label namespace restricted cosigned.sigstore.dev/include=true
```

Quick example to see rejection, 

```shell
pushd $(mktemp -d)

go mod init example.com/demo
cat <<EOF > main.go
package main
import (
  "fmt"
)
func main() {
    fmt.Println("hello world")
}
EOF

demoimage=`ko publish -B example.com/demo --platform=all`
echo Created image $demoimage

popd

kubectl create -n restricted job demo --image=$demoimage
```

Apply out OctoMuffin job,

```shell
ko apply -Bf ./config -- --namespace=restricted
```

And watch the sockeye ui (`kubectl get ksvc sockeye`). 

## Clean up

```shell
kind delete clusters octomuffin
```