package gitlab

import "strings"

const (
	ObjectKindPipeline = "pipeline"
	StatusFailed       = "failed"
)

type PipelineWebhook struct {
	ObjectKind string `json:"object_kind"`

	ObjectAttributes struct {
		ID     int    `json:"id"`
		Status string `json:"status"`
	} `json:"object_attributes"`

	Project struct {
		Name              string `json:"name"`
		PathWithNamespace string `json:"path_with_namespace"`
	} `json:"project"`
}

// IsFailedPipeline reports whether the payload is a failed pipeline event.
func (p PipelineWebhook) IsFailedPipeline() bool {
	return p.ObjectKind == ObjectKindPipeline && p.ObjectAttributes.Status == StatusFailed
}

// UnderGroup reports whether the project belongs to the configured group path.
func (p PipelineWebhook) UnderGroup(groupPath string) bool {
	projectPath := strings.ToLower(strings.TrimSpace(p.Project.PathWithNamespace))
	group := strings.ToLower(strings.Trim(strings.TrimSpace(groupPath), "/"))
	if group == "" || projectPath == "" {
		return false
	}

	return projectPath == group || strings.HasPrefix(projectPath, group+"/")
}
