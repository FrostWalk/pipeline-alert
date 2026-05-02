package gitlab

type PipelineWebhook struct {
	ObjectKind string `json:"object_kind"`

	ObjectAttributes struct {
		Status string `json:"status"`
	} `json:"object_attributes"`

	Project struct {
		Name string `json:"name"`
	} `json:"project"`
}
