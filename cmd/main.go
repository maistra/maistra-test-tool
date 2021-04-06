package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var kubeconfig *string

func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	return os.Getenv("USERPROFILE") // windows
}

func listPods(ns string) {
	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic("config loading error.")
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic("clientset creation error.")
	}

	pods, _ := clientset.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{})
	fmt.Printf("There are %d pods in the %s ns\n", len(pods.Items), ns)
}

func createDeployment(ctx context.Context, ns string) {
	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic("config loading error.")
	}
	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic("clientset creation error.")
	}

	deployment := &appsv1.Deployment{}
	deployment.Name = "example"

	client := clientset.AppsV1().Deployments(ns)
	client.Create(ctx, deployment, metav1.CreateOptions{})
}

func marshalDaemonSet() {
	ds := &appsv1.DaemonSet{}
	ds.Name = "example"  // edit deployment spec

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "    ")
	enc.Encode(ds)
}

func main() {
	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	

}
