package utils

import (
	"context"
	"log"

	nodeNetworkClientSet "github.com/Frank-svg-dev/pam-cni/ClientSet/clientset/versioned"
	nodeNetworkAPI "github.com/Frank-svg-dev/pam-cni/pkg/apis/network.ppnb.io/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
)

func InitNodeNetworkCRClient() *nodeNetworkClientSet.Clientset {
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		log.Fatalf("failed to load kubeconfig: %v", err)
	}
	// 2️⃣ 创建 nodeNetworkClientSet
	nnClient, err := nodeNetworkClientSet.NewForConfig(config)
	if err != nil {
		log.Fatalf("failed to create dynamic client: %v", err)
	}

	return nnClient
}

func InitNodeNetworkCR(nnClient *nodeNetworkClientSet.Clientset, instanceUUID string, hostname string) string {
	nodeNetwork, err := nnClient.NetworkV1alpha1().NodeNetworks().Get(context.Background(), hostname, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			//create NodeNetworkCR
			nodeNetwork, err = nnClient.NetworkV1alpha1().NodeNetworks().Create(context.Background(), &nodeNetworkAPI.NodeNetwork{
				ObjectMeta: metav1.ObjectMeta{
					Name: hostname,
				},
				Spec: nodeNetworkAPI.NodeNetworkSpec{
					InstanceID: instanceUUID,
				},
			}, metav1.CreateOptions{})

			if err != nil {
				log.Fatalf("failed to create NodeNetwork: %v", err)
			}

			return nodeNetwork.Name

		}
		log.Fatalf("failed to get NodeNetwork CR: %v", err)
	}

	if nodeNetwork.Name != hostname {
		log.Fatalf("NodeNetwork CR named %s does not match hostname %s", nodeNetwork.Name, hostname)
	}

	return nodeNetwork.Name
}
