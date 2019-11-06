#!/bin/bash

oc adm policy add-scc-to-user anyuid -z bookinfo-productpage -n bookinfo 
oc adm policy add-scc-to-user anyuid -z bookinfo-reviews -n bookinfo 
oc adm policy add-scc-to-user anyuid -z bookinfo-ratings-v2 -n bookinfo
oc adm policy add-scc-to-user anyuid -z default -n bookinfo

