package utils

import (
	"context"
	"fmt"
	"log"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

func InitKubernetesClient() *dynamic.DynamicClient {
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		log.Fatalf("failed to load kubeconfig: %v", err)
	}
	// 2️⃣ 创建 DynamicClient
	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Fatalf("failed to create dynamic client: %v", err)
	}

	return dynClient
}

func CreateNodeNetworkCR(dynClient *dynamic.DynamicClient, instanceUUID string, hostname string) string {

	gvr := schema.GroupVersionResource{
		Group:    "network.ppnb.io",
		Version:  "v1alpha1",
		Resource: "nodenetworks", // plural 名
	}

	ctx := context.Background()

	obj, err := dynClient.Resource(gvr).Get(context.TODO(), hostname, metav1.GetOptions{})
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			log.Fatalf("get CR failed: %v", err)
		}
	}

	if obj != nil {
		if obj.GetName() == hostname || obj.Object["spec"].(map[string]interface{})["instanceID"] == instanceUUID {
			return hostname
		}
	}

	// 4️⃣ 创建一个 NodeNetwork 对象（Cluster-scoped，无 namespace）
	obj = &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "network.ppnb.io/v1alpha1",
			"kind":       "NodeNetwork",
			"metadata": map[string]interface{}{
				"name": hostname,
			},
			"spec": map[string]interface{}{
				"instanceID": instanceUUID,
			},
		},
	}

	fmt.Println("➡️ Creating NodeNetwork...")
	nn, err := dynClient.Resource(gvr).Create(ctx, obj, metav1.CreateOptions{})
	if err != nil {
		log.Fatalf("create failed: %v", err)
	}
	fmt.Printf("✅ Created: %s\n", nn.GetName())

	return nn.GetName()

	//// 5️⃣ 列出所有 NodeNetwork
	//fmt.Println("➡️ Listing NodeNetworks...")
	//list, err := dynClient.Resource(gvr).List(ctx, metav1.ListOptions{})
	//if err != nil {
	//	log.Fatalf("list failed: %v", err)
	//}
	//
	//for _, item := range list.Items {
	//	name := item.GetName()
	//	spec := item.Object["spec"].(map[string]interface{})
	//	fmt.Printf("- %s => Node=%s, CIDR=%s\n",
	//		name, spec["instanceID"], spec["cidr"])
	//}
	//
	//time.Sleep(20 * time.Second)
	//fmt.Println("➡️ Deleting NodeNetwork...")
	//if err := dynClient.Resource(gvr).Delete(ctx, nn.GetName(), metav1.DeleteOptions{}); err != nil {
	//	log.Fatalf("delete failed: %v", err)
	//}
	//fmt.Println("✅ Deleted NodeNetwork.")
}

func GetNodeCidrNetwork(crName string, dynClient *dynamic.DynamicClient) string {
	gvr := schema.GroupVersionResource{
		Group:    "network.ppnb.io",
		Version:  "v1alpha1",
		Resource: "nodenetworks",
	}

	for i := 0; i < 30; i++ {
		obj, err := dynClient.Resource(gvr).Get(context.TODO(), crName, metav1.GetOptions{})
		if err != nil {
			if strings.Contains(err.Error(), "not found") {

			}
			log.Fatalf("get CR failed: %v", err)
		}

		// ✅ 提取 spec.cidr
		spec, ok := obj.Object["spec"].(map[string]interface{})
		if !ok {
			log.Fatalf("no spec field found")
		}

		cidr, ok := spec["cidr"].(string)
		if !ok {
			log.Println("cidr not found or not string")
			continue
		}

		if cidr != "" {
			return cidr
		}
		continue

	}
	return ""
}
