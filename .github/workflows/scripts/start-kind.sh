#!/bin/bash

# Create controller kind cluster if not present
if [ ! $(kind get clusters | grep controller) ];then
  kind create cluster --name controller --config .github/workflows/scripts/cluster.yaml --image kindest/node:v1.22.7

  # Install Calico calico on controller-cluster
  echo "Installing calico on controller-cluster"
  function wait_for_pods {
    for ns in "$namespace"; do
      for pod in $(kubectl get pods -n $ns | grep -v NAME | awk '{ print $1 }'); do
        counter=0
        echo kubectl get pod $pod -n $ns
        kubectl get pod $pod -n $ns
        while [[ ! ($(kubectl get po $pod -n $ns | grep $pod | awk '{print $3}') =~ ^Running$|^Completed$) ]]; do
          sleep 1
          let counter=counter+1

          if ((counter == $sleep)); then
            echo "POD $pod failed to start in $sleep seconds"
            kubectl get events -n $ns --sort-by='.lastTimestamp'
            echo "Exiting"

            exit -1
          fi
        done
      done
    done
  }

  #install kubectx
  sudo snap install kubectx --classic

  # Switch to Controller cluster...
  kubectx kind-controller

  echo Install the Tigera Calico operator...
  kubectl create -f https://raw.githubusercontent.com/projectcalico/calico/v3.24.1/manifests/tigera-operator.yaml

  echo Download the custom resources necessary to configure Calico
  curl https://raw.githubusercontent.com/projectcalico/calico/v3.24.1/manifests/custom-resources.yaml -O
  sleep 60

  echo Install the custom resource definitions manifest...
  #kubectl create -f https://raw.githubusercontent.com/projectcalico/calico/v3.24.1/manifests/custom-resources.yaml
  kubectl create -f custom-resources.yaml
  sleep 300

  echo "Check for Calico namespaces, pods"
  kubectl get ns
  kubectl get pods -n calico-system
  echo "Wait for Calico to be Running"
  namespace=calico-system
  sleep=900
  wait_for_pods

  kubectl get pods -n calico-system
  
  ip=$(docker inspect controller-control-plane | jq -r '.[0].NetworkSettings.Networks.kind.IPAddress') 
  echo $ip
  # Replace loopback IP with docker ip
  kind get kubeconfig --name controller | sed "s/127.0.0.1.*/$ip:6443/g" > /home/runner/.kube/kind1.yaml
fi

# Create worker1 kind cluster if not present
if [ ! $(kind get clusters | grep worker) ];then
  kind create cluster --name worker --config .github/workflows/scripts/cluster.yaml --image kindest/node:v1.22.7

  # Install Calico calico on worker-cluster
  echo "Installing calico on worker-cluster"
  function wait_for_pods {
    for ns in "$namespace"; do
      for pod in $(kubectl get pods -n $ns | grep -v NAME | awk '{ print $1 }'); do
        counter=0
        echo kubectl get pod $pod -n $ns
        kubectl get pod $pod -n $ns
        while [[ ! ($(kubectl get po $pod -n $ns | grep $pod | awk '{print $3}') =~ ^Running$|^Completed$) ]]; do
          sleep 1
          let counter=counter+1

          if ((counter == $sleep)); then
            echo "POD $pod failed to start in $sleep seconds"
            kubectl get events -n $ns --sort-by='.lastTimestamp'
            echo "Exiting"

            exit -1
          fi
        done
      done
    done
  }

  #install kubectx
  sudo snap install kubectx --classic

  # Switch to Worker cluster...
  kubectx kind-worker

  echo Install the Tigera Calico operator...
  kubectl create -f https://raw.githubusercontent.com/projectcalico/calico/v3.24.1/manifests/tigera-operator.yaml

  echo Download the custom resources necessary to configure Calico
  curl https://raw.githubusercontent.com/projectcalico/calico/v3.24.1/manifests/custom-resources.yaml -O
  sleep 60

  echo Install the custom resource definitions manifest...
  #kubectl create -f https://raw.githubusercontent.com/projectcalico/calico/v3.24.1/manifests/custom-resources.yaml
  kubectl create -f custom-resources.yaml
  sleep 120

  echo "Check for Calico namespaces, pods"
  kubectl get ns
  kubectl get pods -n calico-system
  echo "Wait for Calico to be Running"
  namespace=calico-system
  sleep=900
  wait_for_pods

  kubectl get pods -n calico-system
  
  ip=$(docker inspect worker-control-plane | jq -r '.[0].NetworkSettings.Networks.kind.IPAddress')
  echo $ip
  # Replace loopback IP with docker ip
  kind get kubeconfig --name worker | sed "s/127.0.0.1.*/$ip:6443/g" > /home/runner/.kube/kind2.yaml
fi

KUBECONFIG=/home/runner/.kube/kind1.yaml:/home/runner/.kube/kind2.yaml kubectl config view --raw  > /home/runner/.kube/kinde2e.yaml

if [ ! -f profile/kind.yaml ];then
  # Provide correct IP in kind profile, since worker operator cannot detect internal IP as nodeIp
  IP1=$(docker inspect controller-control-plane | jq -r '.[0].NetworkSettings.Networks.kind.IPAddress')
  IP2=$(docker inspect worker-control-plane | jq -r '.[0].NetworkSettings.Networks.kind.IPAddress')

  cat > profile/kind.yaml << EOF
Kubeconfig: kinde2e.yaml
ControllerCluster:
  Context: kind-controller
  HubChartOptions:
      Repo: "https://kubeslice.github.io/kubeslice"
      SetStrValues:
             "kubeslice.controller.image": "kubeslice-controller"
             "kubeslice.controller.tag": "e2e-latest"
WorkerClusters:
- Context: kind-controller
  NodeIP: ${IP1}
- Context: kind-worker
  NodeIP: ${IP2}
WorkerChartOptions:
  Repo: "https://kubeslice.github.io/kubeslice"
TestSuitesEnabled:
  HubSuite: true
  WorkerSuite: true
EOF

fi
