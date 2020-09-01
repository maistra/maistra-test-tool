#!/bin/bash

oc adm policy add-scc-to-user anyuid -z default -n bookinfo
oc adm policy add-scc-to-user anyuid -z bookinfo-ratings-v2 -n bookinfo
oc adm policy add-scc-to-user anyuid -z httpbin -n bookinfo
oc adm policy add-scc-to-user anyuid -z httpbin -n foo
oc adm policy add-scc-to-user anyuid -z httpbin -n bar
oc adm policy add-scc-to-user anyuid -z httpbin -n legacy
sleep 5