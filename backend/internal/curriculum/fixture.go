package curriculum

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// FixtureGrade represents a single grade's curriculum data from the JSON fixture.
type FixtureGrade struct {
	GradeName     string      `json:"grade_name"`
	LearningAreas []FixtureLA `json:"learning_areas"`
}

// FixtureLA is a learning area within a grade.
type FixtureLA struct {
	Name    string          `json:"name"`
	Code    string          `json:"code"`
	Strands []FixtureStrand `json:"strands"`
}

// FixtureStrand is a strand within a learning area.
type FixtureStrand struct {
	Name       string             `json:"name"`
	SubStrands []FixtureSubStrand `json:"sub_strands"`
}

// FixtureSubStrand is a sub-strand within a strand.
type FixtureSubStrand struct {
	Name     string   `json:"name"`
	Outcomes []string `json:"outcomes"`
}

// CBCFixture is the top-level structure of the curriculum JSON fixture.
type CBCFixture struct {
	Meta   map[string]any `json:"_meta"`
	Grades []FixtureGrade `json:"grades"`
}

// fixturePath returns the absolute path to the CBC curriculum fixture JSON file.
// It discovers the path relative to this source file at runtime so that
// the fixture works regardless of the working directory.
func fixturePath() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		// Fallback: assume running from backend/ dir
		return "internal/database/fixtures/cbc_curriculum.json"
	}
	return filepath.Join(filepath.Dir(filename), "..", "database", "fixtures", "cbc_curriculum.json")
}

// LoadCBCFixture reads and parses the CBC curriculum JSON fixture.
func LoadCBCFixture() (*CBCFixture, error) {
	path := fixturePath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read cbc curriculum fixture %s: %w", path, err)
	}

	var fixture CBCFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		return nil, fmt.Errorf("parse cbc curriculum fixture: %w", err)
	}

	return &fixture, nil
}
