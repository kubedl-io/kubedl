#!/usr/bin/env bash
set -e

echo "#### Running TF test job"
kubectl apply -f example/tf/tf_job_mnist_distributed_simple.yaml

for ((i=1;i<40;i++));
do
  set +e
  PODS=$(kubectl get pod | grep "tf-distributed-simple-worker" | grep "Completed"  | wc -l)
  kubectl logs tf-distributed-simple-worker-0
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
