package api

import "time"

// Pod represents a pod in the API response
type Pod struct {
	// Name is the name of the pod
	Name string `json:"name"`
	// Namespace is the namespace where the pod is running
	Namespace string `json:"namespace"`
	// Status is the current phase of the pod (e.g., Running, Pending)
	Status string `json:"status"`
	// Restarts is the total number of restarts across all containers in the pod
	Restarts int32 `json:"restarts"`
	// Age is the human-readable age of the pod
	Age string `json:"age"`
	// Developer is the name of the developer who owns the pod
	Developer string `json:"developer"`
	// NodeName is the name of the node where the pod is scheduled
	NodeName string `json:"nodeName,omitempty"`
	// PodIP is the IP address of the pod
	PodIP string `json:"podIP,omitempty"`
}

// ListPodsRequest represents query parameters for listing pods
type ListPodsRequest struct {
	Namespace string `json:"namespace,omitempty"`
	Labels    string `json:"labels,omitempty"`
}

// ListPodsResponse represents the response for listing pods
type ListPodsResponse struct {
	Pods  []Pod `json:"pods"`
	Count int   `json:"count"`
}

// DeletePodRequest represents the request to delete a pod
type DeletePodRequest struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

// DeletePodResponse represents the response after deleting a pod
type DeletePodResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// VersionResponse represents the version information
type VersionResponse struct {
	Version   string `json:"version"`
	GitCommit string `json:"gitCommit"`
	BuildTime string `json:"buildTime"`
	GoVersion string `json:"goVersion,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}
