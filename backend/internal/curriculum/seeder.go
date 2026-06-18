package curriculum

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"

	"somotracker/backend/internal/database"
)

// Module is an fx-compatible module for the curriculum seeder.
var Module = fx.Module("curriculum",
	fx.Provide(
		NewSeeder,
	),
)

// Seeder handles seeding CBC curriculum data into the database.
type Seeder struct {
	pool *pgxpool.Pool
}

// NewSeeder creates a new Seeder.
func NewSeeder(pools *database.Pools) *Seeder {
	return &Seeder{pool: pools.PG}
}

// gradeRow holds a grade record fetched from the database.
type gradeRow struct {
	ID   string
	Name string
}

// SeedCBCForSchool populates cbc_learning_areas, cbc_strands, cbc_sub_strands,
// and cbc_learning_outcomes for a newly created CBC school.
// It loads the fixture JSON, looks up grades for the given education system,
// and inserts the curriculum hierarchy in a single transaction.
func (s *Seeder) SeedCBCForSchool(
	ctx context.Context,
	tenantID, schoolID, educationSystemID string,
) error {
	// Load the fixture
	fixture, err := LoadCBCFixture()
	if err != nil {
		return fmt.Errorf("load cbc fixture: %w", err)
	}

	// Fetch all grades for this education system (grade name -> grade UUID)
	grades, err := s.fetchGrades(ctx, educationSystemID)
	if err != nil {
		return fmt.Errorf("fetch grades: %w", err)
	}

	gradeMap := make(map[string]string, len(grades))
	for _, g := range grades {
		gradeMap[g.Name] = g.ID
	}

	// Begin transaction
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	for _, fg := range fixture.Grades {
		gradeID, ok := gradeMap[fg.GradeName]
		if !ok {
			// Skip grades that don't exist in this education system
			continue
		}

		for _, la := range fg.LearningAreas {
			// Insert learning area
			laID, err := s.insertLearningArea(ctx, tx, tenantID, schoolID, educationSystemID, gradeID, la.Name, la.Code)
			if err != nil {
				return fmt.Errorf("insert learning area %q for %s: %w", la.Name, fg.GradeName, err)
			}

			for _, strand := range la.Strands {
				// Insert strand
				strandID, err := s.insertStrand(ctx, tx, laID, strand.Name)
				if err != nil {
					return fmt.Errorf("insert strand %q: %w", strand.Name, err)
				}

				for _, sub := range strand.SubStrands {
					// Insert sub-strand
					subID, err := s.insertSubStrand(ctx, tx, strandID, sub.Name)
					if err != nil {
						return fmt.Errorf("insert sub-strand %q: %w", sub.Name, err)
					}

					for _, outcome := range sub.Outcomes {
						// Insert learning outcome
						if err := s.insertOutcome(ctx, tx, subID, outcome); err != nil {
							return fmt.Errorf("insert outcome: %w", err)
						}
					}
				}
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit curriculum tx: %w", err)
	}

	return nil
}

// fetchGrades retrieves all grades for the given education system.
func (s *Seeder) fetchGrades(ctx context.Context, educationSystemID string) ([]gradeRow, error) {
	const query = `
		SELECT id, name
		FROM grades
		WHERE education_system_id = $1
		ORDER BY sequence_order
	`

	rows, err := s.pool.Query(ctx, query, educationSystemID)
	if err != nil {
		return nil, fmt.Errorf("query grades: %w", err)
	}
	defer rows.Close()

	var grades []gradeRow
	for rows.Next() {
		var g gradeRow
		if err := rows.Scan(&g.ID, &g.Name); err != nil {
			return nil, fmt.Errorf("scan grade: %w", err)
		}
		grades = append(grades, g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return grades, nil
}

// insertLearningArea inserts a single cbc_learning_areas row and returns its ID.
func (s *Seeder) insertLearningArea(
	ctx context.Context,
	tx pgx.Tx,
	tenantID, schoolID, educationSystemID, gradeID, name, code string,
) (string, error) {
	const query = `
		INSERT INTO cbc_learning_areas (tenant_id, school_id, education_system_id, grade_id, name, code)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	var id string
	err := tx.QueryRow(ctx, query,
		tenantID, schoolID, educationSystemID, gradeID, name, code,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("insert learning area: %w", err)
	}
	return id, nil
}

// insertStrand inserts a single cbc_strands row and returns its ID.
func (s *Seeder) insertStrand(ctx context.Context, tx pgx.Tx, learningAreaID, name string) (string, error) {
	const query = `
		INSERT INTO cbc_strands (learning_area_id, name)
		VALUES ($1, $2)
		RETURNING id
	`

	var id string
	err := tx.QueryRow(ctx, query, learningAreaID, name).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("insert strand: %w", err)
	}
	return id, nil
}

// insertSubStrand inserts a single cbc_sub_strands row and returns its ID.
func (s *Seeder) insertSubStrand(ctx context.Context, tx pgx.Tx, strandID, name string) (string, error) {
	const query = `
		INSERT INTO cbc_sub_strands (strand_id, name)
		VALUES ($1, $2)
		RETURNING id
	`

	var id string
	err := tx.QueryRow(ctx, query, strandID, name).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("insert sub-strand: %w", err)
	}
	return id, nil
}

// insertOutcome inserts a single cbc_learning_outcomes row.
func (s *Seeder) insertOutcome(ctx context.Context, tx pgx.Tx, subStrandID, description string) error {
	const query = `
		INSERT INTO cbc_learning_outcomes (sub_strand_id, description)
		VALUES ($1, $2)
	`

	_, err := tx.Exec(ctx, query, subStrandID, description)
	if err != nil {
		return fmt.Errorf("insert learning outcome: %w", err)
	}
	return nil
}
