#!/usr/bin/env bash
set -e

echo "#### Running TF test job"
kubectl apply -f example/tf/tf_job_mnist_distributed_simple.yaml

for ((i=1;i<40;i++));
do
  set +e
  PODS=$(kubectl get pod | grep "tf-distributed-simple-worker" | grep "Completed"  | wc -l)
  KUBEDL_PODNAME=$(kubectl get po -n kubedl-system | grep kubedl | awk  '{print $1}')
  kubectl logs $KUBEDL_PODNAME -n kubedl-system
  set -e
  if [ "$PODS" -eq 3 ]; then
    echo "Succeed: TF distributed test job."
    exit 0
  fi
  echo "waiting for job to complete"
  sleep 5
done

echo "Failed: TF distributed test job."
exit 1
