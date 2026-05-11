package location

type Item struct {
	Code       string `json:"code"`
	FullCode   string `json:"full_code,omitempty"`
	Name       string `json:"name"`
	Level      string `json:"level,omitempty"`
	ParentCode string `json:"parent_code,omitempty"`
}

type ImportStats struct {
	Raw       int
	Provinces int
	Regencies int
	Districts int
	Villages  int
}
