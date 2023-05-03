---
title: "2023q1 PoC Scripts"
linkTitle: "2023q1 PoC Scripts"
weight: 100
---

There are some scripts that automate small steps in the process of
using this PoC.

## Creating SyncTarget/Location pairs

In this PoC, the interface between infrastructure and workload
management is inventory API objects.  Specifically, for each edge
cluster there is a unique pair of SyncTarget and Location objects in a
so-called inventory management workspace.  The following script helps
with making that pair of objects.  Invoke it when your current
workspace is your chosen inventory management workspace.

```console
$ scripts/ensure-location.sh -h
scripts/ensure-location.sh usage: objname labelname=labelvalue...

$ kubectl ws root:imw-1
Current workspace is "root:imw-1".

$ scripts/ensure-location.sh demo1 foo=bar the-word=the-bird
synctarget.workload.kcp.io/demo1 created
location.scheduling.kcp.io/demo1 created
synctarget.workload.kcp.io/demo1 labeled
location.scheduling.kcp.io/demo1 labeled
synctarget.workload.kcp.io/demo1 labeled
location.scheduling.kcp.io/demo1 labeled
```

The above example shows using this script to create a SyncTarget and a
Location named `demo1` with labels `foo=bar` and `the-word=the-bird`.
This was equivalent to the following commands.

```shell
kubectl create -f -<<EOF
apiVersion: workload.kcp.io/v1alpha1
kind: SyncTarget
metadata:
  name: demo1
  labels:
    id: demo1
    foo: bar
    the-word: the-bird
---
apiVersion: scheduling.kcp.io/v1alpha1
kind: Location
metadata:
  name: demo1
  labels:
    foo: bar
    the-word: the-bird
spec:
  resource: {group: workload.kcp.io, version: v1alpha1, resource: synctargets}
  instanceSelector:
    matchLabels: {"id":"demo1"}
EOF
```

This script operates in an idempotent style.  It looks at the current
state and makes whatever changes are needed.  Caveat: it does not cast
a skeptical eye on the spec of a pre-existing Location.

## Removing SyncTarget/Location pairs

The following script undoes whatever remains from a corresponding
usage of `ensure-location.sh`.  Invoke it with the inventory
management workspace current.

```console
$ scripts/remove-location.sh -h
scripts/remove-location.sh usage: objname

$ kubectl ws root:imw-1
Current workspace is "root:imw-1".

$ scripts/remove-location.sh demo1
synctarget.workload.kcp.io "demo1" deleted
location.scheduling.kcp.io "demo1" deleted

$ scripts/remove-location.sh demo1

$ 
```

## Syncer preparation and installation

The syncer runs in each edge cluster and also talks to the
corresponding mailbox workspace.  In order for it to be able to do
that, there is some work to do in the mailbox workspace to create a
ServiceAccount for the syncer to authenticate as and create RBAC
objects to give the syncer the privileges that it needs.  The
following script does those things and also outputs YAML to be used to
install the syncer in the edge cluster.  Invoke this script with the
edge service provider workspace current.

This script assumes that either (a) you have cloned the edge-mc repo
and done `make build` to populate its `bin` directory or (b) you have
fetched a release binary archive and unpacked it to create its `bin`
directory.

```console
$ scripts/mailbox-prep.sh -h
scripts/mailbox-prep.sh usage: (-o file_pathname | --syncer-image container_image_ref )* synctarget_name

$ kubectl ws root:espw
Current workspace is "root:espw".

$ scripts/mailbox-prep.sh demo1
Current workspace is "root:espw:4yqm57kx0m6mn76c-mb-406c54d1-64ce-4fdc-99b3-cef9c4fc5010" (type root:universal).
Creating service account "kcp-edge-syncer-demo1-28at01r3"
Creating cluster role "kcp-edge-syncer-demo1-28at01r3" to give service account "kcp-edge-syncer-demo1-28at01r3"

 1. write and sync access to the synctarget "kcp-edge-syncer-demo1-28at01r3"
 2. write access to apiresourceimports.

Creating or updating cluster role binding "kcp-edge-syncer-demo1-28at01r3" to bind service account "kcp-edge-syncer-demo1-28at01r3" to cluster role "kcp-edge-syncer-demo1-28at01r3".

Wrote physical cluster manifest to demo1-syncer.yaml for namespace "kcp-edge-syncer-demo1-28at01r3". Use

  KUBECONFIG=<pcluster-config> kubectl apply -f "demo1-syncer.yaml"

to apply it. Use

  KUBECONFIG=<pcluster-config> kubectl get deployment -n "kcp-edge-syncer-demo1-28at01r3" kcp-edge-syncer-demo1-28at01r3

to verify the syncer pod is running.
```

Once that script has run, the YAML for the objects to create in the
edge cluster is in your chosen output file.  The default for the
output file is the name of the SyncTarget object with "-syncer.yaml"
appended.

Create those objects with a command like the following; adjust as
needed to configure `kubectl` to modify the edge cluster and read your
chosen output file.

```shell
KUBECONFIG=$demo1_kubeconfig kubectl apply -f demo1-syncer.yaml
```

# Creating a Workload Management Workspace

Such a workspace needs not only to be created but also populated with
an `APIBinding` to the edge API and, if desired, an `APIBinding` to
the Kubernetes containerized workload API.

Invoke this script when the current kcp workspace is the parent of the
desired workload management workspace (WMW).

The default behavior is to include an `APIBinding` to the Kubernetes
containerized workload API, and there are optional command line flags
to control this behavior.

This script works in idempotent style, doing whatever work remains to
be done.

```console
$ scripts/ensure-wmw.sh -h
Usage: kubectl ws parent; scripts/ensure-wmw.sh [--with-kube | --no-kube] wm_workspace_name

$ kubectl ws .
Current workspace is "root:my-org".

$ scripts/ensure-wmw.sh example-wmw
Current workspace is "root".
Current workspace is "root:my-org".
Workspace "example-wmw" (type root:universal) created. Waiting for it to be ready...
Workspace "example-wmw" (type root:universal) is ready to use.
Current workspace is "root:my-org:example-wmw" (type root:universal).
apibinding.apis.kcp.io/bind-espw created
apibinding.apis.kcp.io/bind-kube created

$ kubectl ws ..
Current workspace is "root:my-org".

$ scripts/ensure-wmw.sh example-wmw
Current workspace is "root".
Current workspace is "root:my-org".
Current workspace is "root:my-org:example-wmw" (type root:universal).

$ kubectl ws ..
Current workspace is "root:my-org".

$ scripts/ensure-wmw.sh example-wmw --no-kube
Current workspace is "root".
Current workspace is "root:my-org".
Current workspace is "root:my-org:example-wmw" (type root:universal).
apibinding.apis.kcp.io "bind-kube" deleted

$ 
```

Deleting a WMW is done by simply deleting its `Workspace` object from
the parent.

```console
$ kubectl ws .
Current workspace is "root:my-org:example-wmw".

$ kubectl ws ..
Current workspace is "root:my-org".

$ kubectl delete Workspace example-wmw
workspace.tenancy.kcp.io "example-wmw" deleted

$ 
```