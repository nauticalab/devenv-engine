package api

import "time"

// Pod represents a pod in the API response
type Pod struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Status    string `json:"status"`
	Restarts  int32  `json:"restarts"`
	Age       string `json:"age"`
	Developer string `json:"developer"`
	NodeName  string `json:"nodeName,omitempty"`
	PodIP     string `json:"podIP,omitempty"`
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
