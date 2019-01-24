#!/bin/bash

set -e

export AWS_PROFILE=openshift-dev

./openshift-install destroy cluster --dir=./assets --log-level=debug

