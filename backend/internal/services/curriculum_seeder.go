package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"somotracker/backend/internal/curriculum"
)

type CurriculumSeeder struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

func NewCurriculumSeeder(pool *pgxpool.Pool, logger *zap.Logger) *CurriculumSeeder {
	return &CurriculumSeeder{pool: pool, logger: logger.With(zap.String("service", "curriculum-seeder"))}
}

type subjRow struct {
	id           string
	es           string
	gradeLevelID string
	name         string
	code         string
}
type topicRow struct {
	id   string
	sid  string
	name string
	seq  int
}
type subRow struct {
	id   string
	tid  string
	name string
	seq  int
}

// SeedCurriculumIfEmpty seeds CBE subjects/topics/sub_topics for Kenya CBE if the tables are empty.
// It is idempotent and safe to run on every startup.
func (s *CurriculumSeeder) SeedCurriculumIfEmpty(ctx context.Context) error {
	var count int
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM subjects`).Scan(&count); err != nil {
		return fmt.Errorf("curriculum seed check failed: %w", err)
	}
	if count > 0 {
		s.logger.Info("curriculum already seeded, skipping")
		return nil
	}

	s.logger.Info("curriculum tables empty, seeding CBE curriculum")

	// Resolve Kenya CBE education system and country
	var educationSystemID, countryID string
	err := s.pool.QueryRow(ctx, `
		SELECT es.id, c.id
		FROM education_systems es
		JOIN countries c ON c.id = es.country_id
		WHERE c.country_name = 'Kenya' AND es.system_name ILIKE '%CBE%'
		LIMIT 1
	`).Scan(&educationSystemID, &countryID)
	if err != nil {
		return fmt.Errorf("curriculum seed: resolve education system: %w", err)
	}
	if educationSystemID == "" {
		return fmt.Errorf("curriculum seed: education system not found")
	}

	// Load grade levels for mapping
	gradeMap := make(map[string]string)
	rows, err := s.pool.Query(ctx, `SELECT id, local_label FROM grade_levels WHERE education_system_id = $1`, educationSystemID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var gid, label string
			if rows.Scan(&gid, &label) == nil {
				gradeMap[label] = gid
			}
		}
	}

	subjectsBulk := []subjRow{}
	topicsBulk := []topicRow{}
	subsBulk := []subRow{}

	processCBC := func(fileName string, data []byte) {
		nameOnly := strings.TrimSuffix(fileName, ".json")
		parts := strings.SplitN(nameOnly, ".", 2)
		stem := parts[0]
		gradeLabel := ""
		if strings.HasPrefix(stem, "grade") {
			num := strings.TrimPrefix(stem, "grade")
			gradeLabel = "Grade " + num
		} else if strings.HasPrefix(stem, "pp") {
			gradeLabel = strings.ToUpper(stem)
		}
		gradeID := gradeMap[gradeLabel]

		var subjects []map[string]interface{}
		if err := json.Unmarshal(data, &subjects); err != nil {
			s.logger.Error("curriculum seed: unmarshal failed", zap.String("file", fileName), zap.Error(err))
			return
		}
		for _, sub := range subjects {
			name, _ := sub["name"].(string)
			code, _ := sub["code"].(string)
			if name == "" || code == "" {
				continue
			}
			sid := uuid.New().String()
			subjectsBulk = append(subjectsBulk, subjRow{id: sid, es: educationSystemID, gradeLevelID: gradeID, name: name, code: code})
			strands, ok := sub["strands"].([]interface{})
			if !ok {
				continue
			}
			for idx, s := range strands {
				strand, ok := s.(map[string]interface{})
				if !ok {
					continue
				}
				sName, _ := strand["name"].(string)
				if sName == "" {
					continue
				}
				tid := uuid.New().String()
				topicsBulk = append(topicsBulk, topicRow{id: tid, sid: sid, name: sName, seq: idx + 1})
				subStrands, ok := strand["sub_strands"].([]interface{})
				if !ok {
					continue
				}
				for sIdx, ss := range subStrands {
					sSub, ok := ss.(map[string]interface{})
					if !ok {
						continue
					}
					subName, _ := sSub["name"].(string)
					if subName == "" {
						continue
					}
					ssId := uuid.New().String()
					subsBulk = append(subsBulk, subRow{id: ssId, tid: tid, name: subName, seq: sIdx + 1})
				}
			}
		}
	}

	const embedRoot = "docs/cbc"
	entries, err := curriculum.CbcFS.ReadDir(embedRoot)
	if err != nil {
		return fmt.Errorf("curriculum seed: read embedded dir: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := curriculum.CbcFS.ReadFile(embedRoot + "/" + entry.Name())
		if err != nil {
			s.logger.Error("curriculum seed: read file failed", zap.String("file", entry.Name()), zap.Error(err))
			continue
		}
		processCBC(entry.Name(), data)
	}

	if len(subjectsBulk) > 0 {
		if err := bulkInsertSubjects(ctx, s.pool, subjectsBulk); err != nil {
			return err
		}
	}
	if len(topicsBulk) > 0 {
		if err := bulkInsertTopics(ctx, s.pool, topicsBulk); err != nil {
			return err
		}
	}
	if len(subsBulk) > 0 {
		if err := bulkInsertSubTopics(ctx, s.pool, subsBulk); err != nil {
			return err
		}
	}

	s.logger.Info("curriculum seeding completed", zap.Int("subjects", len(subjectsBulk)), zap.Int("topics", len(topicsBulk)), zap.Int("sub_topics", len(subsBulk)))
	return nil
}

func bulkInsertSubjects(ctx context.Context, pool *pgxpool.Pool, rows []subjRow) error {
	values := make([]string, 0, len(rows))
	args := make([]interface{}, 0, len(rows)*5)
	argIdx := 1
	for _, r := range rows {
		values = append(values, fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,'CORE')", argIdx, argIdx+1, argIdx+2, argIdx+3, argIdx+4))
		gradeVal := interface{}(r.gradeLevelID)
		if r.gradeLevelID == "" {
			gradeVal = nil
		}
		args = append(args, r.id, r.es, gradeVal, r.name, r.code)
		argIdx += 5
	}
	sql := fmt.Sprintf(`INSERT INTO subjects (id, education_system_id, grade_level_id, name, code, type) VALUES %s ON CONFLICT DO NOTHING`, strings.Join(values, ","))
	_, err := pool.Exec(ctx, sql, args...)
	return err
}

func bulkInsertTopics(ctx context.Context, pool *pgxpool.Pool, rows []topicRow) error {
	values := make([]string, 0, len(rows))
	args := make([]interface{}, 0, len(rows)*4)
	argIdx := 1
	for _, r := range rows {
		values = append(values, fmt.Sprintf("($%d,$%d,$%d,$%d)", argIdx, argIdx+1, argIdx+2, argIdx+3))
		args = append(args, r.id, r.sid, r.name, r.seq)
		argIdx += 4
	}
	sql := fmt.Sprintf(`INSERT INTO topics (id, subject_id, name, sequence_index) VALUES %s ON CONFLICT DO NOTHING`, strings.Join(values, ","))
	_, err := pool.Exec(ctx, sql, args...)
	return err
}

func bulkInsertSubTopics(ctx context.Context, pool *pgxpool.Pool, rows []subRow) error {
	values := make([]string, 0, len(rows))
	args := make([]interface{}, 0, len(rows)*4)
	argIdx := 1
	for _, r := range rows {
		values = append(values, fmt.Sprintf("($%d,$%d,$%d,$%d)", argIdx, argIdx+1, argIdx+2, argIdx+3))
		args = append(args, r.id, r.tid, r.name, r.seq)
		argIdx += 4
	}
	sql := fmt.Sprintf(`INSERT INTO sub_topics (id, topic_id, name, sequence_index) VALUES %s ON CONFLICT DO NOTHING`, strings.Join(values, ","))
	_, err := pool.Exec(ctx, sql, args...)
	return err
}
