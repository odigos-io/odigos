#!/bin/bash

# some of the rollouts during the test are triggered by patching an annotation on the workload.
# the annotation value is a timestamp. This timestamp has a resolution of seconds, so we need to wait
# at least 1 second before we can be sure that the rollout has been triggered.
#
# The controller itself could check this timestamp and wait for the next second to trigger the rollout,
# however for now we decided to not implement it as it seems extremely unlikely that this would happen in a real-world scenario.
#
# this is also consistent with the behavior of `kubectl rollout restart deployment ...` which also has a resolution of seconds.
# trying to do a few consecutive `kubectl rollout restart deployment ...` may result in an error such as
# "if restart has already been triggered within the past second, please wait before attempting to trigger another"
sleep 1

kubectl rollout status deployment -l app=coupon
kubectl rollout status deployment -l app=frontend
kubectl rollout status deployment -l app=inventory
kubectl rollout status deployment -l app=pricing
kubectl rollout status deployment -l app=membership