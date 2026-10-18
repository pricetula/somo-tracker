package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL env var required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	// Simple backfill: for each student, read metadata JSON and copy known keys to columns
	// This is a one-off migration. Run with caution.
	rows, err := pool.Query(ctx, `SELECT student_id, metadata FROM students WHERE metadata IS NOT NULL`)
	if err != nil {
		log.Fatalf("query failed: %v", err)
	}
	defer rows.Close()

	updated := 0
	for rows.Next() {
		var studentID string
		var metadata []byte
		if err := rows.Scan(&studentID, &metadata); err != nil {
			log.Printf("scan error: %v", err)
			continue
		}
		var data map[string]interface{}
		if err := json.Unmarshal(metadata, &data); err != nil {
			log.Printf("json unmarshal %s: %v", studentID, err)
			continue
		}
		// Build update with COALESCE to only set if column is null
		// For simplicity, we update via direct SQL with values from metadata if present
		address, _ := data["address"].(string)
		phone, _ := data["phone"].(string)
		email, _ := data["email"].(string)
		nationality, _ := data["nationality"].(string)
		ecName, _ := data["emergency_contact_name"].(string)
		ecPhone, _ := data["emergency_contact_phone"].(string)
		specialNeeds, _ := data["special_needs"].(string)

		// Only update if at least one field present
		if address == "" && phone == "" && email == "" && nationality == "" && ecName == "" && ecPhone == "" && specialNeeds == "" {
			continue
		}
		_, err := pool.Exec(ctx, `
			UPDATE students SET
				address = COALESCE(NULLIF($1, ''), address),
				phone = COALESCE(NULLIF($2, ''), phone),
				email = COALESCE(NULLIF($3, ''), email),
				nationality = COALESCE(NULLIF($4, ''), nationality),
				emergency_contact_name = COALESCE(NULLIF($5, ''), emergency_contact_name),
				emergency_contact_phone = COALESCE(NULLIF($6, ''), emergency_contact_phone),
				special_needs = COALESCE(NULLIF($7, ''), special_needs)
			WHERE student_id = $8
		`, address, phone, email, nationality, ecName, ecPhone, specialNeeds, studentID)
		if err != nil {
			log.Printf("update failed %s: %v", studentID, err)
			continue
		}
		updated++
	}
	fmt.Printf("Backfill complete, updated %d students\n", updated)
}
