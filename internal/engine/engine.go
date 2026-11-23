// Package engine scores coverage. It does not read files and it does not estimate headroom.
package engine

import (
	"fmt"
	"sort"
	"time"

	"github.com/Harry0052/gridatlas/internal/model"
)

// Build scores every atlas region at a frozen as-of date.
func Build(bundle *model.Bundle, asOf time.Time) (*model.Atlas, error) {
	if bundle == nil {
		return nil, fmt.Errorf("bundle is required")
	}
	asOf = time.Date(asOf.Year(), asOf.Month(), asOf.Day(), 0, 0, 0, 0, time.UTC)
	if asOf.IsZero() {
		return nil, fmt.Errorf("as-of date is required")
	}
	regions := append([]model.Region(nil), bundle.Regions...)
	sort.Slice(regions, func(i, j int) bool { return regions[i].ID < regions[j].ID })

	atlas := &model.Atlas{
		SchemaVersion: model.AtlasSchema,
		AsOf:          asOf.Format("2006-01-02"),
		Generator:     model.Generator,
		Limitations:   model.Limitations(),
		Regions:       make([]model.RegionResult, 0, len(regions)),
	}
	for _, region := range regions {
		atlas.Regions = append(atlas.Regions, model.RegionResult{
			RegionID: region.ID,
			Name:     region.Name,
			Metrics: model.Metrics{
				InstalledNameplateMW: scoreProduct(bundle, region, model.ProductInstalled, asOf),
				QueuedRequestedMW:    scoreProduct(bundle, region, model.ProductQueued, asOf),
				HeadroomMW:           headroom(),
			},
		})
	}
	return atlas, nil
}

func headroom() model.Headroom {
	return model.Headroom{
		Status:        model.StatusUnobserved,
		DisplayStatus: model.StatusUnobserved,
		ValueMW:       nil,
		Zero:          false,
		Unit:          model.UnitMW,
		Definition:    model.Definition("headroom"),
		Reason:        model.ReasonNotEstimated,
	}
}

func scoreProduct(bundle *model.Bundle, region model.Region, product string, asOf time.Time) model.Metric {
	metric := blankMetric(product, asOf)
	var row *model.ManifestRow
	for i := range bundle.Manifest {
		item := &bundle.Manifest[i]
		if item.Region == region.ID && item.Product == product {
			row = item
			break
		}
	}
	if row == nil {
		metric.Status = model.StatusMissing
		metric.DisplayStatus = model.StatusMissing
		metric.Reason = model.ReasonNoSource
		return metric
	}
	applyProvenance(&metric, row, asOf)
	rows := append([]model.Measurement(nil), bundle.Snapshots[row.SnapshotFile]...)
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Line == rows[j].Line {
			return rows[i].RecordID < rows[j].RecordID
		}
		return rows[i].Line < rows[j].Line
	})

	join, joinOK := findCrosswalk(bundle, row)
	if !joinOK {
		metric.Status = model.StatusMissing
		metric.DisplayStatus = model.DisplayUnknown
		metric.Reason = model.ReasonNoDefensibleMapping
		metric.Trace = traceAll(rows, "not_joined")
		return metric
	}
	if !join.Supported {
		metric.Status = model.StatusIncompatible
		metric.DisplayStatus = model.StatusIncompatible
		metric.Reason = model.ReasonUnsupportedJoin
		metric.Trace = traceAll(rows, "not_joined")
		return metric
	}
	age := ageDays(row.ReleaseDate, asOf)
	metric.AgeDays = &age
	if age < 0 {
		metric.Status = model.StatusIncompatible
		metric.DisplayStatus = model.StatusIncompatible
		metric.Reason = model.ReasonReleaseAfterAsOf
		metric.Trace = traceAll(rows, "not_joined")
		return metric
	}
	if row.Unit != model.UnitMW || anyUnit(rows, func(unit string) bool { return unit != model.UnitMW }) {
		metric.Status = model.StatusIncompatible
		metric.DisplayStatus = model.StatusIncompatible
		metric.Reason = model.ReasonMislabeledUnit
		metric.SourceUnit = row.Unit
		metric.Trace = traceAll(rows, "rejected_unit")
		return metric
	}
	wantBasis := model.BasisNameplate
	if product == model.ProductQueued {
		wantBasis = model.BasisRequested
	}
	if anyBasis(rows, wantBasis) {
		metric.Status = model.StatusIncompatible
		metric.DisplayStatus = model.StatusIncompatible
		metric.Reason = model.ReasonIncompatibleBasis
		metric.Trace = traceAll(rows, "rejected_basis")
		return metric
	}
	if badStatus(product, rows) {
		metric.Status = model.StatusIncompatible
		metric.DisplayStatus = model.StatusIncompatible
		metric.Reason = model.ReasonIncompatibleStatus
		metric.Trace = traceAll(rows, "rejected_status")
		return metric
	}
	if product == model.ProductQueued && missingActiveRequestID(rows) {
		metric.Status = model.StatusIncompatible
		metric.DisplayStatus = model.StatusIncompatible
		metric.Reason = model.ReasonMissingRequestID
		metric.Trace = traceAll(rows, "rejected_status")
		return metric
	}

	var sum int
	var included int
	if product == model.ProductInstalled {
		sum, included = aggregateInstalled(&metric, rows)
	} else {
		sum, included = aggregateQueued(&metric, rows)
	}
	if included == 0 {
		metric.Status = model.StatusMissing
		metric.DisplayStatus = model.StatusMissing
		metric.Reason = model.ReasonNoIncludedRecords
		metric.ValueMW = nil
		return metric
	}
	if sum > model.MaxAggregateMW {
		metric.Status = model.StatusIncompatible
		metric.DisplayStatus = model.StatusIncompatible
		metric.Reason = model.ReasonAggregateOutOfBounds
		metric.ValueMW = nil
		return metric
	}
	metric.ValueMW = &sum
	metric.Zero = sum == 0
	metric.Status = model.StatusAvailable
	metric.DisplayStatus = model.StatusAvailable
	if age > row.FreshnessThresholdDays {
		metric.Status = model.StatusStale
		metric.DisplayStatus = model.StatusStale
	}
	return metric
}

func blankMetric(product string, asOf time.Time) model.Metric {
	return model.Metric{
		Unit:       model.UnitMW,
		Definition: model.Definition(product),
		Coverage: model.Coverage{
			IncludedRecordIDs: []string{},
		},
		Provenance: model.Provenance{
			AsOf:        asOf.Format("2006-01-02"),
			InputSchema: model.InputSchema,
		},
		Trace: []model.TraceItem{},
	}
}

func applyProvenance(metric *model.Metric, row *model.ManifestRow, asOf time.Time) {
	threshold := row.FreshnessThresholdDays
	metric.ReleaseDate = row.ReleaseDate.Format("2006-01-02")
	metric.ReportingPeriod = row.ReportingPeriod
	metric.FreshnessThresholdDays = &threshold
	metric.SourceUnit = row.Unit
	metric.Provenance = model.Provenance{
		Product:         row.Product,
		SourceURL:       row.SourceURL,
		ReleaseDate:     row.ReleaseDate.Format("2006-01-02"),
		ReportingPeriod: row.ReportingPeriod,
		MappingMethod:   row.MappingMethod,
		ContentSHA256:   row.ContentSHA256,
		SnapshotFile:    row.SnapshotFile,
		AsOf:            asOf.Format("2006-01-02"),
		InputSchema:     model.InputSchema,
	}
}

func findCrosswalk(bundle *model.Bundle, row *model.ManifestRow) (model.Crosswalk, bool) {
	for _, item := range bundle.Crosswalk {
		if item.SourceRegion == row.Region && item.AtlasRegion == row.Region && item.MappingMethod == row.MappingMethod {
			return item, true
		}
	}
	return model.Crosswalk{}, false
}

func anyUnit(rows []model.Measurement, pred func(string) bool) bool {
	for _, row := range rows {
		if pred(row.Unit) {
			return true
		}
	}
	return false
}

func anyBasis(rows []model.Measurement, want string) bool {
	for _, row := range rows {
		if row.Basis != want {
			return true
		}
	}
	return false
}

func badStatus(product string, rows []model.Measurement) bool {
	for _, row := range rows {
		if product == model.ProductInstalled && row.RowStatus != "operating" {
			return true
		}
		if product == model.ProductQueued {
			switch row.RowStatus {
			case "active", "withdrawn", "omitted", "ambiguous":
			default:
				return true
			}
		}
	}
	return false
}

func missingActiveRequestID(rows []model.Measurement) bool {
	for _, row := range rows {
		if row.RowStatus == "active" && row.RequestID == "" {
			return true
		}
	}
	return false
}

func aggregateInstalled(metric *model.Metric, rows []model.Measurement) (int, int) {
	type group struct {
		qty  map[int]struct{}
		rows []model.Measurement
	}
	groups := map[string]*group{}
	var order []string
	for _, row := range rows {
		item := groups[row.RecordID]
		if item == nil {
			item = &group{qty: map[int]struct{}{}}
			groups[row.RecordID] = item
			order = append(order, row.RecordID)
		}
		item.qty[row.Quantity] = struct{}{}
		item.rows = append(item.rows, row)
	}
	sum := 0
	included := 0
	for _, id := range order {
		item := groups[id]
		if len(item.qty) != 1 {
			metric.Coverage.AmbiguousRequests++
			for _, row := range item.rows {
				metric.Trace = append(metric.Trace, traceOf(row, "ambiguous"))
			}
			continue
		}
		var qty int
		for value := range item.qty {
			qty = value
		}
		sum += qty
		included++
		metric.Coverage.IncludedRecords++
		metric.Coverage.IncludedRecordIDs = append(metric.Coverage.IncludedRecordIDs, id)
		for i, row := range item.rows {
			disposition := "included"
			if i > 0 {
				disposition = "duplicate"
				metric.Coverage.DuplicateRowsIgnored++
			}
			metric.Trace = append(metric.Trace, traceOf(row, disposition))
		}
	}
	sort.Strings(metric.Coverage.IncludedRecordIDs)
	return sum, included
}

func aggregateQueued(metric *model.Metric, rows []model.Measurement) (int, int) {
	type group struct {
		qty  map[int]struct{}
		rows []model.Measurement
	}
	groups := map[string]*group{}
	var order []string
	disposition := map[int]string{}
	for _, row := range rows {
		switch row.RowStatus {
		case "withdrawn":
			metric.Coverage.WithdrawnRows++
			disposition[row.Line] = "withdrawn"
			continue
		case "omitted":
			metric.Coverage.OmittedRows++
			disposition[row.Line] = "omitted"
			continue
		case "ambiguous":
			metric.Coverage.AmbiguousRows++
			disposition[row.Line] = "ambiguous"
			continue
		}
		item := groups[row.RequestID]
		if item == nil {
			item = &group{qty: map[int]struct{}{}}
			groups[row.RequestID] = item
			order = append(order, row.RequestID)
		}
		item.qty[row.Quantity] = struct{}{}
		item.rows = append(item.rows, row)
	}
	sum := 0
	included := 0
	for _, id := range order {
		item := groups[id]
		if len(item.qty) != 1 {
			metric.Coverage.AmbiguousRequests++
			for _, row := range item.rows {
				disposition[row.Line] = "ambiguous"
			}
			continue
		}
		var qty int
		for value := range item.qty {
			qty = value
		}
		sum += qty
		included++
		metric.Coverage.IncludedRecords++
		metric.Coverage.IncludedRecordIDs = append(metric.Coverage.IncludedRecordIDs, item.rows[0].RecordID)
		for i, row := range item.rows {
			if i == 0 {
				disposition[row.Line] = "included"
				continue
			}
			disposition[row.Line] = "duplicate"
			metric.Coverage.DuplicateRowsIgnored++
		}
	}
	for _, row := range rows {
		metric.Trace = append(metric.Trace, traceOf(row, disposition[row.Line]))
	}
	sort.Strings(metric.Coverage.IncludedRecordIDs)
	return sum, included
}

func traceAll(rows []model.Measurement, disposition string) []model.TraceItem {
	out := make([]model.TraceItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, traceOf(row, disposition))
	}
	return out
}

func traceOf(row model.Measurement, disposition string) model.TraceItem {
	return model.TraceItem{
		RecordID:    row.RecordID,
		RequestID:   row.RequestID,
		Technology:  row.Technology,
		QuantityMW:  row.Quantity,
		Unit:        row.Unit,
		Disposition: disposition,
	}
}

func ageDays(release, asOf time.Time) int {
	r := time.Date(release.Year(), release.Month(), release.Day(), 0, 0, 0, 0, time.UTC)
	a := time.Date(asOf.Year(), asOf.Month(), asOf.Day(), 0, 0, 0, 0, time.UTC)
	return int(a.Sub(r).Hours() / 24)
}
