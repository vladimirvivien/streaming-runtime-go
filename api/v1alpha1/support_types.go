package v1alpha1

// StreamSetup defines stream data selection, composition, filter and output
type StreamSetup struct {
	From []string       `json:"from"`
	To   []OutputTarget `json:"to"`
	// +optional
	Select string `json:"select"`
	// +optional
	Where string `json:"where"`
}

type OutputTarget struct {
	// +optional
	Stream string `json:"stream"`
	// +optional
	Component string `json:"component"`
}
