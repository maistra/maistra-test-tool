#!/bin/bash

echo "This script is used for cleaning everything of a single instance oc cluster."
echo

set -x

oc cluster down
for i in $(mount | grep openshift | awk '{ print $3}'); do sudo umount "$i"; done

ls openshift.local.clusterup && sudo rm -rf openshift.local.clusterup
