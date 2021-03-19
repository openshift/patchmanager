# patch-manager

This tool is used to automate OpenShift z-stream patch manager daily routine by listing all z-stream candidate pull requests and triaging
them based on multiple criteria.

Triage is done via scoring system where each pull request is being classified by the following:

* **Bug Severity**
* **Bug Flags**
* **PM Score**
* **Component Priority**

Complete implementation and details can be found in `pkg/classifiers` package.

Each classifier can be configured via `config.yaml` file. Most of them assign "score" to each pull request. The score is additive, so in the example
below, an "urgent" bug with *TestBlocker* flag will get score `1.8` which will likely put it at the top of the list.

#### Example config.yaml:

```yaml
---
# MergeWindow describe a time window when pull requests can be cherry-picked for the z-stream.
# TODO: This is not implemented yet.
mergeWindow:
  from:
  to:
# Capacity describe the QE capacity for the "next" week per QE group.
capacity:
  default: 5 # <- this is a "default" capacity if no capacity is specified for a component
  groups:
    - name: API&Auth
      capacity: 5 # <- this is the QE capacity for all components listed below
      components:
      - Master
      - apiserver-auth
      - authentication
      - service-ca
      - openshift-apiserver
      - oauth-apiserver
      - kube-apiserver
      - oauth-proxy
      - kube-storage-version-migrator
      - config-operator
    - name: Workloads
      capacity: 5
      components:
      - Deployments
      - Command Line Interface
      - oc
      - kube-controller-manager
      - kube-scheduler
# Classifiers describe how much score points a single pull request should get. (0-1)
# Score impact the position of a PR in merge queue.
classifiers:
  # Flags classifier assign score based on bugzilla flags present in bug associated with pull request
  flags:
    "TestBlocker": 0.8
    "UpgradeBlocker": 0.8
    "Security": 0.5
  # Components classifier assign score based on importance/criticality of components
  components:
    "authentication": 0.5
    "networking": 0.5
    "node": 0.5
    "kube-apiserver": 0.5
  # Severities classifier assign score based on bug Severity field
  severities:
    "urgent": 1.0
    "high": 0.5
    "medium": 0.2
    "low": 0.1
    "unknown": -1.0
  # PMScores classifier assign score based on PMScore field ranges
  pmScores:
    - from: 0
      to: 30
      score: 0
    - from: 30
      to: 50
      score: 0.2
    - from: 50
      to: 100
      score: 0.5
    - from: 100
      to: 999
      score: 0.8
```


Using the above config, only 2 pull requests for *Networking* component will be approved and only 5 pull requests for *kube-apiserver* bugzilla
component will be picked.


## Pre-requirements

* You will need [Github Personal Token](https://github.com/settings/tokens) exported via environment variable `GITHUB_TOKEN` (or use command flag).
* You will need [Bugzilla API Token](https://bugzilla.redhat.com/userprefs.cgi?tab=apikey) exported via environment variable `BUGZILLA_APIKEY` (or use command flag).
* You need a config file (check examples/ directory). You can also use `PATCHMANAGER_CONFIG=https://raw.githubusercontent.com/openshift/patchmanager/main/examples/config.yaml`

## Installation

To install this command run:

```
$ go get github.com/mfojtik/patch-manager/cmd/patchmanager
```

The binary `patchmanager` should be installed in your *$GOPATH/bin* directory.

### Development

In case you want to contribute, you can use *Makefile* to build the `patchmanager` binary by invoking `make`.

## Usage

1. `patchmanager run --release=4.x --config=path/to/config.yaml -o candidates.yaml` will produce YAML file of candidate pull request for *4.x* release already sorted
  and scored based on the classifiers. The `capacity` flag will cause that on *N* pull requests will be "picked".
  
*Example YAML file:*
  
```yaml
items:
- pullRequest:
    # Description: MCDDrainError "Drain failed on  , updates may be blocked" missing
    # rendered node name
    # Bug: https://bugzilla.redhat.com/show_bug.cgi?id=1906298
    # Component: Machine Config Operator
    # Severity: high
    # PM Score 52
    url: https://github.com/openshift/machine-config-operator/pull/2419
    decision: pick
    score: 1.0
- pullRequest:
    # Description: 4.7 to 4.6 downgrade fails due to 4.7 Cluster Profile Support manifest
    # changes
    # Bug: https://bugzilla.redhat.com/show_bug.cgi?id=1925199
    # Component: Cluster Version Operator
    # Severity: medium
    # PM Score 77
    url: https://github.com/openshift/cluster-version-operator/pull/512
    decision: pick
    score: 0.7
- pullRequest:
    # Description: [Kuryr] Available port count not correctly calculated for alerts
    # Bug: https://bugzilla.redhat.com/show_bug.cgi?id=1897526
    # Component: Networking
    # Severity: low
    # PM Score 0
    url: https://github.com/openshift/cluster-network-operator/pull/907
    decision: skip
    score: 0.6
    decisionReason: target capacity for component Networking is 2
```

2. A human patch manager need to review this YAML file and make decisions on individual changes. Decision can be either **pick** or **skip**.
   
3. Once you are done editing YAML file, you can run the `patchmanager approve -f candidates.yaml` command which will apply the `cherry-pick-approved` label
  on ALL pull requests with "pick" decision.
   
4. Alternatively, you can use `patchmanager list -f candidates.yaml` to format the pull requests in human readable table:

```console
$ patchmanager list -f candidates.yaml 
  URL (35)                                                                              SCORE   DECISION   REASON                                                      
 ------------------------------------------------------------------------------------- ------- ---------- ------------------------------------------------------------ 
  https://github.com/openshift/ovn-kubernetes/pull/460                                   2.00   pick                                                                   
  https://github.com/openshift/machine-config-operator/pull/2462                         1.50   pick                                                                   
  https://github.com/openshift/machine-config-operator/pull/2426                         1.50   pick                                                                   
  https://github.com/openshift/openshift-apiserver/pull/187                              1.00   pick                                                                   
  https://github.com/openshift/cluster-ingress-operator/pull/570                         0.20   skip       maximum picks set by patch manager for this z-stream is 10  
  https://github.com/openshift/origin/pull/25913                                         0.20   skip       maximum picks set by patch manager for this z-stream is 10  
  https://github.com/operator-framework/operator-lifecycle-manager/pull/2036             0.20   skip       maximum picks set by patch manager for this z-stream is 10  
  https://github.com/openshift/console-operator/pull/512                                 0.20   skip       maximum picks set by patch manager for this z-stream is 10  
```