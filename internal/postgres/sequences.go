package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type sequenceSample struct {
	SchemaName   string
	SequenceName string
	SampleTime   time.Time
	LastValue    int64
}

type SequenceForwardResult struct {
	SchemaName     string
	SequenceName   string
	OldValue       int64
	NewValue       int64
	IncreasedBy    int64
	ProjectedValue int64
}

type FastForwardResult struct {
	Sequences        []SequenceForwardResult
	SampleDuration   time.Duration
	BufferMultiplier float64
	SkipSeconds      int
	TotalForwarded   int
	TotalSkipped     int
}

func sampleSequenceValues(ctx context.Context, db *sql.DB) ([]sequenceSample, error) {
	sampleTime := time.Now()

	rows, err := db.QueryContext(ctx, `
		SELECT
			schemaname,
			sequencename,
			last_value
		FROM pg_sequences
		WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
		ORDER BY schemaname, sequencename
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to sample sequences: %w", err)
	}
	defer rows.Close()

	var samples []sequenceSample
	for rows.Next() {
		sample := sequenceSample{SampleTime: sampleTime}
		var lastValue sql.NullInt64
		err := rows.Scan(&sample.SchemaName, &sample.SequenceName, &lastValue)
		if err != nil {
			return nil, fmt.Errorf("failed to scan sequence sample: %w", err)
		}
		if lastValue.Valid {
			sample.LastValue = lastValue.Int64
		}
		samples = append(samples, sample)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sequence samples: %w", err)
	}

	return samples, nil
}

// FastForwardSequences samples source sequences twice to estimate write rates,
// then advances destination sequences to accommodate future writes.
func FastForwardSequences(ctx context.Context, srcDB, destDB *sql.DB, sampleDuration time.Duration, bufferMultiplier float64, skipSeconds int) (*FastForwardResult, error) {
	result := &FastForwardResult{
		SampleDuration:   sampleDuration,
		BufferMultiplier: bufferMultiplier,
		SkipSeconds:      skipSeconds,
	}

	// Take first sample
	sample1, err := sampleSequenceValues(ctx, srcDB)
	if err != nil {
		return nil, fmt.Errorf("failed to take first sample: %w", err)
	}

	// Wait for sample duration
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(sampleDuration):
	}

	// Take second sample
	sample2, err := sampleSequenceValues(ctx, srcDB)
	if err != nil {
		return nil, fmt.Errorf("failed to take second sample: %w", err)
	}

	// Build maps for quick lookup
	sample1Map := make(map[string]*sequenceSample)
	for i := range sample1 {
		s := &sample1[i]
		key := fmt.Sprintf("%s.%s", s.SchemaName, s.SequenceName)
		sample1Map[key] = s
	}

	// Calculate rates and fast-forward
	for _, s2 := range sample2 {
		key := fmt.Sprintf("%s.%s", s2.SchemaName, s2.SequenceName)
		s1, ok := sample1Map[key]
		if !ok {
			continue
		}

		// Calculate the rate of change
		elapsed := s2.SampleTime.Sub(s1.SampleTime).Seconds()
		if elapsed <= 0 {
			elapsed = sampleDuration.Seconds()
		}

		diff := s2.LastValue - s1.LastValue
		if diff < 0 {
			diff = 0
		}

		// Calculate projected value with buffer
		ratePerSecond := float64(diff) / elapsed
		projectedIncrement := int64(ratePerSecond * float64(skipSeconds) * bufferMultiplier)
		projectedValue := s2.LastValue + projectedIncrement

		// Get current destination value
		var destValue int64
		err := destDB.QueryRowContext(ctx, fmt.Sprintf(
			"SELECT last_value FROM %s.%s",
			QuoteIdentifier(s2.SchemaName),
			QuoteIdentifier(s2.SequenceName),
		)).Scan(&destValue)
		if err != nil {
			// Sequence might not exist on destination, skip it
			result.TotalSkipped++
			continue
		}

		// Only fast-forward if the projected value is greater
		if projectedValue <= destValue {
			result.TotalSkipped++
			continue
		}

		// Set the sequence to the projected value (escape single quotes for regclass literal)
		_, err = destDB.ExecContext(ctx, fmt.Sprintf(
			"SELECT setval('%s', $1, true)",
			escapeSequenceRegclass(s2.SchemaName, s2.SequenceName),
		), projectedValue)
		if err != nil {
			return nil, fmt.Errorf("failed to fast-forward sequence %s.%s: %w", s2.SchemaName, s2.SequenceName, err)
		}

		result.Sequences = append(result.Sequences, SequenceForwardResult{
			SchemaName:     s2.SchemaName,
			SequenceName:   s2.SequenceName,
			OldValue:       destValue,
			NewValue:       projectedValue,
			IncreasedBy:    projectedValue - destValue,
			ProjectedValue: projectedValue,
		})
		result.TotalForwarded++
	}

	return result, nil
}

// escapeSequenceRegclass returns a safe schema.sequence string for use inside a regclass literal.
func escapeSequenceRegclass(schema, sequence string) string {
	escape := func(s string) string { return strings.ReplaceAll(s, "'", "''") }
	return escape(schema) + "." + escape(sequence)
}
