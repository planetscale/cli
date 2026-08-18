package metrics

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/dustin/go-humanize"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

type seriesSummaryRow struct {
	Metric     string `header:"metric"`
	Series     string `header:"series"`
	Dimensions string `header:"dimensions"`
	Latest     string `header:"latest"`
	Min        string `header:"min"`
	Avg        string `header:"avg"`
	Max        string `header:"max"`
	Trend      string `header:"trend"`
}

type metricPointCSVRow struct {
	Timestamp string  `csv:"timestamp"`
	Metric    string  `csv:"metric"`
	Label     string  `csv:"label"`
	Labels    string  `csv:"labels"`
	Value     float64 `csv:"value"`
}

type instantHumanRow struct {
	Metric     string `header:"metric"`
	Dimensions string `header:"dimensions"`
	Value      string `header:"value"`
}

type instantCSVRow struct {
	Metric     string `csv:"metric"`
	Label      string `csv:"label"`
	Dimensions string `csv:"dimensions"`
	Value      string `csv:"value"`
}

func printSeriesSummary(ch *cmdutil.Helper, database, branch string, response *ps.MetricSeries) error {
	if len(response.Series) == 0 {
		ch.Printer.Printf("No metric series returned for %s in %s.\n",
			printer.BoldBlue(branch), printer.BoldBlue(database))
		return nil
	}

	maxPoints := 0
	for _, series := range response.Series {
		if len(series.Points) > maxPoints {
			maxPoints = len(series.Points)
		}
	}

	ch.Printer.Printf("Metrics for %s/%s\n", printer.BoldBlue(database), printer.BoldBlue(branch))
	ch.Printer.Printf("Range: %s · interval: %ds · up to %d points/series\n\n",
		formatMetricRange(response.StartDate, response.EndDate), response.Interval, maxPoints)

	return ch.Printer.PrintResource(seriesSummaryRows(response))
}

func seriesSummaryRows(response *ps.MetricSeries) []*seriesSummaryRow {
	rows := make([]*seriesSummaryRow, 0, len(response.Series))
	for _, series := range response.Series {
		values := pointValues(series.Points)
		if len(values) == 0 {
			rows = append(rows, &seriesSummaryRow{
				Metric:     humanMetricName(series.Metric),
				Series:     seriesName(series),
				Dimensions: formatLabels(series.Labels),
				Latest:     "n/a",
				Min:        "n/a",
				Avg:        "n/a",
				Max:        "n/a",
				Trend:      "—",
			})
			continue
		}

		min, avg, max := valueStats(values)
		rows = append(rows, &seriesSummaryRow{
			Metric:     humanMetricName(series.Metric),
			Series:     seriesName(series),
			Dimensions: formatLabels(series.Labels),
			Latest:     formatMetricValue(series.Metric, values[len(values)-1]),
			Min:        formatMetricValue(series.Metric, min),
			Avg:        formatMetricValue(series.Metric, avg),
			Max:        formatMetricValue(series.Metric, max),
			Trend:      sparkline(values, 12),
		})
	}
	return rows
}

func metricPointRows(response *ps.MetricSeries) []*metricPointCSVRow {
	rows := make([]*metricPointCSVRow, 0)
	for _, series := range response.Series {
		labels, _ := json.Marshal(series.Labels)
		for _, point := range series.Points {
			if len(point) < 2 {
				continue
			}
			rows = append(rows, &metricPointCSVRow{
				Timestamp: time.Unix(int64(point[0]), 0).UTC().Format(time.RFC3339),
				Metric:    series.Metric,
				Label:     series.Label,
				Labels:    string(labels),
				Value:     point[1],
			})
		}
	}
	return rows
}

func instantMetricHumanRows(response *ps.InstantMetrics) []*instantHumanRow {
	rows := make([]*instantHumanRow, 0)
	for _, metric := range response.Metrics {
		name := humanMetricName(metric.Label)
		if metric.Label == "" {
			name = humanMetricName(metric.Metric)
		}
		for _, value := range metric.Values {
			rows = append(rows, &instantHumanRow{
				Metric:     name,
				Dimensions: formatInstantDimensions(value, "—"),
				Value:      formatInstantValue(metric.Metric, value["value"]),
			})
		}
	}
	return rows
}

func instantMetricCSVRows(response *ps.InstantMetrics) []*instantCSVRow {
	rows := make([]*instantCSVRow, 0)
	for _, metric := range response.Metrics {
		for _, value := range metric.Values {
			rows = append(rows, &instantCSVRow{
				Metric:     metric.Metric,
				Label:      metric.Label,
				Dimensions: formatInstantDimensions(value, ""),
				Value:      formatRawValue(value["value"]),
			})
		}
	}
	return rows
}

func pointValues(points [][]float64) []float64 {
	values := make([]float64, 0, len(points))
	for _, point := range points {
		if len(point) < 2 || math.IsNaN(point[1]) || math.IsInf(point[1], 0) {
			continue
		}
		values = append(values, point[1])
	}
	return values
}

func valueStats(values []float64) (float64, float64, float64) {
	min, max := values[0], values[0]
	var sum float64
	for _, value := range values {
		min = math.Min(min, value)
		max = math.Max(max, value)
		sum += value
	}
	return min, sum / float64(len(values)), max
}

func sparkline(values []float64, width int) string {
	if len(values) == 0 || width <= 0 {
		return "—"
	}

	samples := values
	if len(values) > width {
		samples = make([]float64, width)
		for i := range samples {
			index := int(math.Round(float64(i) * float64(len(values)-1) / float64(width-1)))
			samples[i] = values[index]
		}
	}

	min, _, max := valueStats(samples)
	levels := []rune("▁▂▃▄▅▆▇█")
	var result strings.Builder
	for _, value := range samples {
		level := 3
		if max != min {
			level = int(math.Round((value - min) / (max - min) * float64(len(levels)-1)))
		}
		result.WriteRune(levels[level])
	}
	return result.String()
}

func formatMetricRange(start, end time.Time) string {
	localStart, localEnd := start.Local(), end.Local()
	if localStart.YearDay() == localEnd.YearDay() && localStart.Year() == localEnd.Year() {
		return fmt.Sprintf("%s–%s", localStart.Format("2006-01-02 15:04"), localEnd.Format("15:04 MST"))
	}
	return fmt.Sprintf("%s–%s", localStart.Format("2006-01-02 15:04 MST"), localEnd.Format("2006-01-02 15:04 MST"))
}

func seriesName(series *ps.TimeSeries) string {
	if series.Label == "" || normalizeName(series.Label) == normalizeName(humanMetricName(series.Metric)) {
		return "—"
	}
	return series.Label
}

func formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return "—"
	}
	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, labels[key]))
	}
	return strings.Join(parts, ", ")
}

func formatInstantDimensions(value map[string]any, empty string) string {
	keys := make([]string, 0, len(value))
	for key := range value {
		if key != "value" && value[key] != nil && fmt.Sprint(value[key]) != "" {
			keys = append(keys, key)
		}
	}
	if len(keys) == 0 {
		return empty
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", key, value[key]))
	}
	return strings.Join(parts, ", ")
}

func humanMetricName(name string) string {
	name = strings.TrimPrefix(name, "planetscale_")
	name = strings.TrimSpace(strings.ReplaceAll(name, "_", " "))
	if name == "" {
		return "Metric"
	}

	words := strings.Fields(name)
	replacements := map[string]string{
		"cpu":          "CPU",
		"iops":         "IOPS",
		"oom":          "OOM",
		"pgbouncer":    "PgBouncer",
		"postgres":     "PostgreSQL",
		"rss":          "RSS",
		"util":         "utilization",
		"percentages":  "percentage",
		"vtgate":       "VTGate",
		"vreplication": "VReplication",
		"wal":          "WAL",
	}
	for i, word := range words {
		if replacement, ok := replacements[strings.ToLower(word)]; ok {
			words[i] = replacement
		}
	}
	if _, ok := replacements[strings.ToLower(words[0])]; !ok {
		runes := []rune(words[0])
		runes[0] = unicode.ToUpper(runes[0])
		words[0] = string(runes)
	}
	return strings.Join(words, " ")
}

func normalizeName(name string) string {
	return strings.Join(strings.Fields(strings.ToLower(name)), " ")
}

type unit int

const (
	unitNumber unit = iota
	unitBytes
	unitBytesPerSecond
	unitPercent
	unitMilliseconds
	unitSeconds
)

func metricUnit(metric string) unit {
	lower := strings.ToLower(metric)
	if strings.Contains(lower, "bytes") && strings.HasSuffix(lower, "_rate") {
		return unitBytesPerSecond
	}
	if strings.Contains(lower, "bytes") || lower == "storage_per_table" || lower == "shard_storage_usage" || lower == "shard_storage_available" || lower == "planetscale_primary_storage_usage" {
		return unitBytes
	}
	if strings.Contains(lower, "percent") || strings.Contains(lower, "cpu_by_az") || strings.Contains(lower, "memory_by_az") || strings.HasSuffix(lower, "cpu_usage") || strings.HasSuffix(lower, "memory_usage") || lower == "block_cache_hit_ratio" {
		return unitPercent
	}
	if strings.Contains(lower, "latency") || strings.Contains(lower, "duration_millis") {
		return unitMilliseconds
	}
	if strings.Contains(lower, "lag") || strings.HasSuffix(lower, "_seconds") || strings.HasSuffix(lower, "_age_succeeded") {
		return unitSeconds
	}
	return unitNumber
}

func formatMetricValue(metric string, value float64) string {
	switch metricUnit(metric) {
	case unitBytes:
		if value >= 0 {
			return humanize.IBytes(uint64(math.Round(value)))
		}
	case unitBytesPerSecond:
		if value >= 0 {
			return humanize.IBytes(uint64(math.Round(value))) + "/s"
		}
	case unitPercent:
		return formatSignificant(value, 4) + "%"
	case unitMilliseconds:
		return formatSignificant(value, 4) + " ms"
	case unitSeconds:
		return formatSignificant(value, 4) + " s"
	}
	return formatNumber(value)
}

func formatInstantValue(metric string, value any) string {
	if number, ok := numericValue(value); ok {
		return formatMetricValue(metric, number)
	}
	return formatRawValue(value)
}

func numericValue(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		f, err := v.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

func formatRawValue(value any) string {
	if number, ok := numericValue(value); ok {
		return strconv.FormatFloat(number, 'f', -1, 64)
	}
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func formatNumber(value float64) string {
	if math.Abs(value-math.Round(value)) < 1e-9 && value <= math.MaxInt64 && value >= math.MinInt64 {
		return humanize.Comma(int64(math.Round(value)))
	}
	return formatSignificant(value, 4)
}

func formatSignificant(value float64, digits int) string {
	if value == 0 || digits <= 0 {
		return "0"
	}

	magnitude := int(math.Floor(math.Log10(math.Abs(value))))
	decimals := digits - magnitude - 1
	if decimals < 0 {
		decimals = 0
	}
	if decimals > 12 {
		decimals = 12
	}

	scale := math.Pow10(decimals)
	rounded := math.Round(value*scale) / scale
	formatted := humanize.CommafWithDigits(rounded, decimals)
	if strings.Contains(formatted, ".") {
		formatted = strings.TrimRight(strings.TrimRight(formatted, "0"), ".")
	}
	return formatted
}
