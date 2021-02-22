# patch-manager

This tool is used to automate OpenShift z-stream patch manager daily routine by listing all z-stream candidate pull requests and triaging
them based on multiple criteria.

Triage is done via scoring system where each pull request is being classified by the following:

* **Bug Severity**
* **Bug Flags**
* **PM Score**
* **Component**

Complete implementation and details can be found in `pkg/classifier` package.

## Installation

To install this command, simple `go install github.com/mfojtik/patch-manager/cmd/patchmanager` should be sufficient.

### Development

In case you want to contribute, you can use *Makefile* to build the patchmanager binary by running `make`.

## Pre-requirements

* You will need [Github Personal Token](https://github.com/settings/tokens) exported via environment variable `GITHUB_TOKEN` (or use command flag).
* You will need [Bugzilla API Token](https://bugzilla.redhat.com/userprefs.cgi?tab=apikey) exported via environment variable `BUGZILLA_APIKEY` (or use command flag).

## Usage

* `patchmanager run --release=4.x --capacity=N -o candidates.yaml` will produce YAML file of candidate pull request for *4.x* release already sorted
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
    # Score: 1.00
    url: https://github.com/openshift/machine-config-operator/pull/2419
    decision: pick
- pullRequest:
    # Description: 4.7 to 4.6 downgrade fails due to 4.7 Cluster Profile Support manifest
    # changes
    # Bug: https://bugzilla.redhat.com/show_bug.cgi?id=1925199
    # Component: Cluster Version Operator
    # Severity: medium
    # PM Score 77
    # Score: 0.70
    url: https://github.com/openshift/cluster-version-operator/pull/512
    decision: pick

```

* A human patch manager need to review this YAML file and make decisions on individual changes. Decision can be either **pick** or **skip**.

* Once you are done editing YAML file, you can run the `patchmanager approve -f candidates.yaml` command which will apply the `cherry-pick-approved` label
  on ALL pull requests with "pick" decision.