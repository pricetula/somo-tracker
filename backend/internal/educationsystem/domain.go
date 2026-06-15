package educationsystem

// EducationSystem represents a curriculum framework a school operates under.
type EducationSystem struct {
	ID          string `db:"id"           json:"id"`
	Name        string `db:"name"         json:"name"`
	CountryCode string `db:"country_code" json:"country_code"`
}
