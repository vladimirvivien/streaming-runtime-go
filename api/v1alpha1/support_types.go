package v1alpha1

// DataSelection defines data selection expressions to select and filter out streamed events
type DataSelection struct {
	// +optional
	Data string `json:"data"`
	// +optional
	Where string `json:"where"`
}

type OutputTarget struct {
	// +optional
	Stream string `json:"stream"`
	// +optional
	Component string `json:"component"`
}
