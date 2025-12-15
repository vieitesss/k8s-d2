package kube

import (
	"flag"
	"io"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"
)

type Client struct {
	clientset *kubernetes.Clientset
}

func init() {
	// Suppress klog output (used by k8s client library)
	klog.SetOutput(io.Discard)
	klog.LogToStderr(false)

	// Prevent klog from adding flags
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	klog.InitFlags(fs)
}

func NewClient(kubeconfigPath string) (*Client, error) {
	if kubeconfigPath == "" {
		if home := homedir.HomeDir(); home != "" {
			kubeconfigPath = filepath.Join(home, ".kube", "config")
		}
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &Client{clientset: clientset}, nil
}
