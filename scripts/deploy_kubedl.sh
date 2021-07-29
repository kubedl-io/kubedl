#!/usr/bin/env bash

if [ -z "$IMG" ]; then
  echo "no found IMG env"
  exit 1
fi

set -e

echo "#### applying CRDs ####"
kubectl apply -f config/crd/bases

echo "#### applying all_in_one.yaml ####"
kubectl apply -f config/manager/all_in_one.yaml