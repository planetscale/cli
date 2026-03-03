package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// SequenceInfo contains information about a PostgreSQL sequence.
type SequenceInfo struct {
	SchemaName   string
	SequenceName string
	DataType     string
	StartValue   int64
	MinValue     int64
	MaxValue     int64
	IncrementBy  int64
	CycleOption  bool
	CacheSize    int64
	LastValue    int64
	OwnerTable   string
	OwnerColumn  string
}

// SequenceSample contains sampled sequence values.
type SequenceSample struct {
	SchemaName   string
	SequenceName string
	SampleTime   time.Time
	LastValue    int64
}

// SequenceForwardResult contains the result of fast-forwarding a sequence.
type SequenceForwardResult struct {
	SchemaName     string
	SequenceName   string
	OldValue       int64
	NewValue       int64
	IncreasedBy    int64
	ProjectedValue int64
}

// FastForwardResult contains the overall result of fast-forwarding sequences.
type FastForwardResult struct {
	Sequences        []SequenceForwardResult
	SampleDuration   time.Duration
	BufferMultiplier float64
	SkipSeconds      int
	TotalForwarded   int
	TotalSkipped     int
}

// GetSequences retrieves all sequences in the database.
func GetSequences(ctx context.Context, db *sql.DB) ([]SequenceInfo, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT
			schemaname,
			sequencename,
			data_type,
			start_value,
			min_value,
			max_value,
			increment_by,
			cycle,
			cache_size,
			last_value
		FROM pg_sequences
		WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
		ORDER BY schemaname, sequencename
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get sequences: %w", err)
	}
	defer rows.Close()

	var sequences []SequenceInfo
	for rows.Next() {
		var seq SequenceInfo
		var lastValue sql.NullInt64
		err := rows.Scan(
			&seq.SchemaName,
			&seq.SequenceName,
			&seq.DataType,
			&seq.StartValue,
			&seq.MinValue,
			&seq.MaxValue,
			&seq.IncrementBy,
			&seq.CycleOption,
			&seq.CacheSize,
			&lastValue,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan sequence: %w", err)
		}
		if lastValue.Valid {
			seq.LastValue = lastValue.Int64
		}
		sequences = append(sequences, seq)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sequences: %w", err)
	}

	// Get owner information for each sequence
	for i := range sequences {
		seq := &sequences[i]
		err := db.QueryRowContext(ctx, `
			SELECT
				COALESCE(a.attrelid::regclass::text, ''),
				COALESCE(a.attname, '')
			FROM pg_class c
			JOIN pg_namespace n ON n.oid = c.relnamespace
			LEFT JOIN pg_depend d ON d.objid = c.oid AND d.deptype = 'a'
			LEFT JOIN pg_attribute a ON a.attrelid = d.refobjid AND a.attnum = d.refobjsubid
			WHERE n.nspname = $1 AND c.relname = $2 AND c.relkind = 'S'
		`, seq.SchemaName, seq.SequenceName).Scan(&seq.OwnerTable, &seq.OwnerColumn)
		if err != nil && err != sql.ErrNoRows {
			// Non-critical error, continue without owner info
		}
	}

	return sequences, nil
}

// SampleSequenceValues samples the current values of all sequences.
func SampleSequenceValues(ctx context.Context, db *sql.DB) ([]SequenceSample, error) {
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

	var samples []SequenceSample
	for rows.Next() {
		sample := SequenceSample{SampleTime: sampleTime}
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

// FastForwardSequences fast-forwards sequences on the destination based on source values.
// It samples the source sequences twice with a delay to estimate write rates, then
// fast-forwards the destination sequences to accommodate future writes.
func FastForwardSequences(ctx context.Context, srcDB, destDB *sql.DB, sampleDuration time.Duration, bufferMultiplier float64, skipSeconds int) (*FastForwardResult, error) {
	result := &FastForwardResult{
		SampleDuration:   sampleDuration,
		BufferMultiplier: bufferMultiplier,
		SkipSeconds:      skipSeconds,
	}

	// Take first sample
	sample1, err := SampleSequenceValues(ctx, srcDB)
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
	sample2, err := SampleSequenceValues(ctx, srcDB)
	if err != nil {
		return nil, fmt.Errorf("failed to take second sample: %w", err)
	}

	// Build maps for quick lookup
	sample1Map := make(map[string]*SequenceSample)
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

		// Set the sequence to the projected value
		_, err = destDB.ExecContext(ctx, fmt.Sprintf(
			"SELECT setval('%s.%s', $1, true)",
			s2.SchemaName,
			s2.SequenceName,
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

// SetSequenceValue sets a sequence to a specific value.
func SetSequenceValue(ctx context.Context, db *sql.DB, schemaName, sequenceName string, value int64, isCalled bool) error {
	_, err := db.ExecContext(ctx, fmt.Sprintf(
		"SELECT setval('%s.%s', $1, $2)",
		schemaName,
		sequenceName,
	), value, isCalled)
	if err != nil {
		return fmt.Errorf("failed to set sequence value: %w", err)
	}
	return nil
}

// RestartSequence restarts a sequence from its start value.
func RestartSequence(ctx context.Context, db *sql.DB, schemaName, sequenceName string) error {
	_, err := db.ExecContext(ctx, fmt.Sprintf(
		"ALTER SEQUENCE %s.%s RESTART",
		QuoteIdentifier(schemaName),
		QuoteIdentifier(sequenceName),
	))
	if err != nil {
		return fmt.Errorf("failed to restart sequence: %w", err)
	}
	return nil
}

// GetSequenceValue gets the current value of a sequence without incrementing it.
func GetSequenceValue(ctx context.Context, db *sql.DB, schemaName, sequenceName string) (int64, error) {
	var value int64
	err := db.QueryRowContext(ctx, fmt.Sprintf(
		"SELECT last_value FROM %s.%s",
		QuoteIdentifier(schemaName),
		QuoteIdentifier(sequenceName),
	)).Scan(&value)
	if err != nil {
		return 0, fmt.Errorf("failed to get sequence value: %w", err)
	}
	return value, nil
}
