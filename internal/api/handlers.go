package api

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/nauticalab/devenv-engine/internal/auth"
	"github.com/nauticalab/devenv-engine/internal/k8s"
	corev1 "k8s.io/api/core/v1"
)

// Handler holds dependencies for HTTP handlers
type Handler struct {
	// k8sClient is the Kubernetes client for interacting with the cluster
	k8sClient *k8s.Client
	// version is the application version
	version string
	// gitCommit is the git commit hash of the build
	gitCommit string
	// buildTime is the time when the application was built
	buildTime string
	// goVersion is the Go version used to build the application
	goVersion string
}

// NewHandler creates a new Handler instance
func NewHandler(k8sClient *k8s.Client, version, gitCommit, buildTime, goVersion string) *Handler {
	return &Handler{
		k8sClient: k8sClient,
		version:   version,
		gitCommit: gitCommit,
		buildTime: buildTime,
		goVersion: goVersion,
	}
}

// Health handles GET /api/v1/health
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	respondSuccess(w, HealthResponse{
		Status:    "ok",
		Timestamp: time.Now(),
	})
}

// Version handles GET /api/v1/version
func (h *Handler) Version(w http.ResponseWriter, r *http.Request) {
	respondSuccess(w, VersionResponse{
		Version:   h.version,
		GitCommit: h.gitCommit,
		BuildTime: h.buildTime,
		GoVersion: h.goVersion,
	})
}

// ListPods handles GET /api/v1/pods
// Lists pods filtered by the authenticated developer
func (h *Handler) ListPods(w http.ResponseWriter, r *http.Request) {
	// Get authenticated developer from context
	developer, ok := auth.GetDeveloperFromContext(r.Context())
	if !ok {
		respondForbidden(w, "No developer identity found")
		return
	}

	// Get query parameters
	namespace := r.URL.Query().Get("namespace")
	if namespace == "" {
		namespace = "default"
	}

	// Build label selector to filter by developer
	labelSelector := fmt.Sprintf("developer=%s", developer)

	log.Printf("Listing pods for developer=%s in namespace=%s with labels=%s", developer, namespace, labelSelector)

	// List pods from Kubernetes
	podList, err := h.k8sClient.ListPodsWithLabels(r.Context(), namespace, labelSelector)
	if err != nil {
		log.Printf("Error listing pods: %v", err)
		respondInternalError(w, "Failed to list pods")
		return
	}

	// Convert to API response format
	pods := make([]Pod, 0, len(podList.Items))
	for _, p := range podList.Items {
		pods = append(pods, convertPodToAPI(&p))
	}

	respondSuccess(w, ListPodsResponse{
		Pods:  pods,
		Count: len(pods),
	})
}

// DeletePod handles DELETE /api/v1/pods/{namespace}/{name}
// Deletes a pod only if it belongs to the authenticated developer
func (h *Handler) DeletePod(w http.ResponseWriter, r *http.Request) {
	// Get authenticated developer from context
	developer, ok := auth.GetDeveloperFromContext(r.Context())
	if !ok {
		respondForbidden(w, "No developer identity found")
		return
	}

	// Get path parameters
	namespace := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	if namespace == "" || name == "" {
		respondBadRequest(w, "Missing namespace or pod name")
		return
	}

	log.Printf("Delete request for pod %s/%s by developer=%s", namespace, name, developer)

	// First, get the pod to check if it belongs to this developer
	pod, err := h.k8sClient.GetPodByName(r.Context(), namespace, name)
	if err != nil {
		log.Printf("Error getting pod %s/%s: %v", namespace, name, err)
		respondNotFound(w, fmt.Sprintf("Pod %s/%s not found", namespace, name))
		return
	}

	// Check if pod belongs to this developer
	podDeveloper := pod.Labels["developer"]
	if podDeveloper != developer {
		log.Printf("Authorization failed: pod developer=%s, authenticated developer=%s", podDeveloper, developer)
		respondForbidden(w, "You can only delete your own pods")
		return
	}

	// Delete the pod
	if err := h.k8sClient.DeletePod(r.Context(), namespace, name); err != nil {
		log.Printf("Error deleting pod %s/%s: %v", namespace, name, err)
		respondInternalError(w, "Failed to delete pod")
		return
	}

	log.Printf("Successfully deleted pod %s/%s by developer=%s", namespace, name, developer)

	respondSuccess(w, DeletePodResponse{
		Success: true,
		Message: fmt.Sprintf("Pod %s/%s deleted successfully", namespace, name),
	})
}

// convertPodToAPI converts a Kubernetes pod to the API Pod type
func convertPodToAPI(pod *corev1.Pod) Pod {
	// Calculate age
	age := formatAge(pod.CreationTimestamp.Time)

	// Calculate total restarts
	restarts := int32(0)
	for _, cs := range pod.Status.ContainerStatuses {
		restarts += cs.RestartCount
	}

	// Get status
	status := string(pod.Status.Phase)
	if pod.DeletionTimestamp != nil {
		status = "Terminating"
	}

	// Get developer label
	developer := pod.Labels["developer"]
	if developer == "" {
		developer = "-"
	}

	return Pod{
		Name:      pod.Name,
		Namespace: pod.Namespace,
		Status:    status,
		Restarts:  restarts,
		Age:       age,
		Developer: developer,
		NodeName:  pod.Spec.NodeName,
		PodIP:     pod.Status.PodIP,
	}
}

// formatAge returns a human-readable duration string
func formatAge(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	}
	if duration < time.Hour {
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	}
	if duration < 24*time.Hour {
		return fmt.Sprintf("%dh", int(duration.Hours()))
	}
	days := int(duration.Hours() / 24)
	return fmt.Sprintf("%dd", days)
}

// WhoAmI handles GET /api/v1/auth/whoami
// Returns the identity of the authenticated user
func (h *Handler) WhoAmI(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.GetIdentityFromContext(r.Context())
	if !ok {
		respondUnauthorized(w, "Not authenticated")
		return
	}

	respondSuccess(w, WhoAmIResponse{
		Identity: *identity,
	})
}

// WhoAmIResponse represents the response for the WhoAmI endpoint
type WhoAmIResponse struct {
	auth.Identity
}
