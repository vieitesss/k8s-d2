package k8sddcmd

// DiagramArgs builds the k8sdd diagram command used by the Dagger module.
func DiagramArgs(namespace string, allNamespaces bool, outputPath string, includeStorage bool, imageOutput bool) []string {
	args := []string{"k8sdd", "diagram"}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}
	if allNamespaces {
		args = append(args, "--all-namespaces")
	}
	if includeStorage {
		args = append(args, "--include-storage")
	}
	if imageOutput {
		return append(args, "--image", outputPath)
	}

	return append(args, "-o", outputPath)
}
