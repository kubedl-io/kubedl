#!/usr/bin/env bash

if [ -z "$IMG_TAG" ]; then
  echo "no found $IMG_TAG env"
  exit 1
fi

set -e

echo "#### applying CRDs ####"
kubectl apply -f config/crd/bases

echo "#### applying all_in_one.yaml ####"
sed -i -E "s/(.*image: docker.io\/kubedl\/kubedl:).*/\1$IMG_TAG/g" config/manager/all_in_one.yaml
sed -i "s/IfNotPresent/Never/g" config/manager/all_in_one.yaml
sed -i "/        resources:/,+6d" config/manager/all_in_one.yaml
kubectl apply -f config/manager/all_in_one.yaml