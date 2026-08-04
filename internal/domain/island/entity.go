package island

type Island struct {
	Code         string   `json:"code"`
	ProvinceCode string   `json:"province_code"`
	Name         string   `json:"name"`
	Latitude     *float64 `json:"latitude,omitempty"`
	Longitude    *float64 `json:"longitude,omitempty"`
	Status       string   `json:"status,omitempty"`
	Area         *float64 `json:"area,omitempty"`
	Notes        string   `json:"notes,omitempty"`
}

type Pagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type Page struct {
	Items      []Island   `json:"items"`
	Pagination Pagination `json:"pagination"`
}
