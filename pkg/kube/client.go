package kube

import (
	"flag"
	"io"
	"path/filepath"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"
)

type Client struct {
	clientset kubernetes.Interface
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

// ServiceTargetPort returns the rendered targetPort value, preserving named ports
// and defaulting omitted targetPort values to the service port itself.
func ServiceTargetPort(port corev1.ServicePort) string {
	if port.TargetPort.StrVal != "" {
		return port.TargetPort.StrVal
	}
	if port.TargetPort.IntVal != 0 {
		return strconv.FormatInt(int64(port.TargetPort.IntVal), 10)
	}
	return strconv.FormatInt(int64(port.Port), 10)
}
