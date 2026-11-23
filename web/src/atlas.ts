export interface Coverage {
  included_records: number;
  duplicate_rows_ignored: number;
  omitted_rows: number;
  withdrawn_rows: number;
  ambiguous_rows: number;
  ambiguous_requests: number;
  included_record_ids: string[];
}

export interface TraceItem {
  record_id: string;
  request_id?: string;
  technology?: string;
  quantity_mw: number;
  unit?: string;
  disposition: string;
}

export interface Metric {
  status: string;
  display_status: string;
  value_mw: number | null;
  zero: boolean;
  unit: string;
  source_unit?: string;
  definition: string;
  release_date?: string;
  reporting_period?: string;
  freshness_threshold_days?: number;
  age_days?: number | null;
  reason?: string;
  coverage: Coverage;
  provenance: {
    source_url?: string;
    snapshot_file?: string;
    content_sha256?: string;
    mapping_method?: string;
    as_of: string;
  };
  trace: TraceItem[];
}

export interface Headroom {
  status: string;
  display_status: string;
  value_mw: number | null;
  zero: boolean;
  unit: string;
  definition: string;
  reason: string;
}

export interface RegionResult {
  region_id: string;
  name: string;
  metrics: {
    installed_nameplate_mw: Metric;
    queued_requested_mw: Metric;
    headroom_mw: Headroom;
  };
}

export interface Atlas {
  schema_version: string;
  as_of: string;
  limitations: string[];
  regions: RegionResult[];
}

export interface Geo {
  type: string;
  features: Array<{
    properties: { region_id: string; boundary_note?: string };
    geometry: { coordinates: number[][][] };
  }>;
}

export type MetricKey = "installed_nameplate_mw" | "queued_requested_mw" | "headroom_mw";

export interface ViewRow {
  regionId: string;
  name: string;
  status: string;
  displayStatus: string;
  valueMW: number | null;
  zero: boolean;
  releaseDate: string;
  definition: string;
  reason: string;
  sourceUnit: string;
  coverage: string;
  snapshot: string;
  sourceURL: string;
  sha256: string;
  trace: TraceItem[];
}

const labels: Record<MetricKey, string> = {
  installed_nameplate_mw: "Installed nameplate MW",
  queued_requested_mw: "Queued requested MW",
  headroom_mw: "Unobserved headroom",
};

export function metricLabel(metric: MetricKey): string {
  return labels[metric];
}

export function viewRows(atlas: Atlas, metric: MetricKey): ViewRow[] {
  return atlas.regions.map((region) => {
    if (metric === "headroom_mw") {
      const cell = region.metrics.headroom_mw;
      return {
        regionId: region.region_id,
        name: region.name,
        status: cell.status,
        displayStatus: cell.display_status,
        valueMW: cell.value_mw,
        zero: cell.zero,
        releaseDate: "",
        definition: cell.definition,
        reason: cell.reason,
        sourceUnit: "",
        coverage: "not applicable",
        snapshot: "",
        sourceURL: "",
        sha256: "",
        trace: [],
      };
    }
    const cell = region.metrics[metric];
    return {
      regionId: region.region_id,
      name: region.name,
      status: cell.status,
      displayStatus: cell.display_status,
      valueMW: cell.value_mw,
      zero: cell.zero,
      releaseDate: cell.release_date ?? "",
      definition: cell.definition,
      reason: cell.reason ?? "",
      sourceUnit: cell.source_unit ?? "",
      coverage: coverageText(cell.coverage),
      snapshot: cell.provenance.snapshot_file ?? "",
      sourceURL: cell.provenance.source_url ?? "",
      sha256: cell.provenance.content_sha256 ?? "",
      trace: cell.trace,
    };
  });
}

export function valueText(row: ViewRow): string {
  if (row.valueMW === null || row.valueMW === undefined) {
    if (row.displayStatus === "unknown") return "unknown";
    if (row.displayStatus === "unobserved") return "unobserved";
    if (row.displayStatus === "missing") return "missing";
    if (row.displayStatus === "incompatible") return "incompatible";
    return row.displayStatus;
  }
  if (row.zero) return `${row.valueMW} MW (reported zero)`;
  return `${row.valueMW} MW`;
}

export function coverageText(coverage: Coverage): string {
  return `included ${coverage.included_records}, duplicate rows ${coverage.duplicate_rows_ignored}, omitted ${coverage.omitted_rows}, withdrawn ${coverage.withdrawn_rows}, ambiguous requests ${coverage.ambiguous_requests}`;
}

export function announcement(metric: MetricKey, rows: ViewRow[]): string {
  const counts = new Map<string, number>();
  for (const row of rows) {
    counts.set(row.displayStatus, (counts.get(row.displayStatus) ?? 0) + 1);
  }
  const parts = [...counts.entries()].map(([status, count]) => `${count} ${status}`);
  return `${metricLabel(metric)}. ${parts.join(", ")}.`;
}

export function regionAnnouncement(metric: MetricKey, row: ViewRow): string {
  const date = row.releaseDate ? `, released ${row.releaseDate}` : "";
  return `${metricLabel(metric)}, ${row.name}, ${valueText(row)}, ${row.displayStatus}${date}.`;
}

export function traceText(row: ViewRow): string {
  const lines = [
    `${row.name}`,
    `Status: ${row.displayStatus}`,
    `Value: ${valueText(row)}`,
    `Definition: ${row.definition}`,
  ];
  if (row.reason) lines.push(`Reason: ${row.reason}`);
  if (row.sourceUnit) lines.push(`Source unit: ${row.sourceUnit}`);
  if (row.releaseDate) lines.push(`Release date: ${row.releaseDate}`);
  if (row.snapshot) lines.push(`Snapshot: ${row.snapshot}`);
  if (row.sourceURL) lines.push(`Source: ${row.sourceURL}`);
  if (row.sha256) lines.push(`SHA-256: ${row.sha256}`);
  lines.push(`Coverage: ${row.coverage}`);
  if (row.trace.length === 0) {
    lines.push("No source rows were included.");
  } else {
    lines.push("Rows:");
    for (const item of row.trace) {
      const request = item.request_id ? ` request ${item.request_id}` : "";
      lines.push(`${item.record_id}${request} ${item.technology ?? ""} ${item.quantity_mw} ${item.unit ?? "MW"} ${item.disposition}`);
    }
  }
  return lines.join("\n");
}

export function mount(doc: Document, atlas: Atlas, geo: Geo, summary: string): void {
  const limits = doc.getElementById("limits");
  if (limits) {
    for (const line of atlas.limitations) {
      const item = doc.createElement("li");
      item.textContent = line;
      limits.appendChild(item);
    }
  }
  const live = doc.getElementById("live");
  const body = doc.getElementById("table-body");
  const trace = doc.getElementById("trace");
  const map = doc.getElementById("map");
  const caption = doc.getElementById("table-caption");
  let metric: MetricKey = "installed_nameplate_mw";
  let selected = atlas.regions[0]?.region_id ?? "";

  const draw = (focusTarget: "map" | "table" | "none" = "none") => {
    const rows = viewRows(atlas, metric);
    if (caption) caption.textContent = `${metricLabel(metric)} as of ${atlas.as_of}. Other metrics stay in the export.`;
    if (live) live.textContent = announcement(metric, rows);
    drawTable(doc, body, rows, selected, (regionId) => {
      selected = regionId;
      draw("table");
      const row = rows.find((item) => item.regionId === regionId);
      if (live && row) live.textContent = regionAnnouncement(metric, row);
    });
    drawMap(doc, map, geo, rows, selected, (regionId) => {
      selected = regionId;
      draw("map");
      const row = rows.find((item) => item.regionId === regionId);
      if (live && row) live.textContent = regionAnnouncement(metric, row);
    });
    const current = rows.find((item) => item.regionId === selected) ?? rows[0];
    if (trace && current) trace.textContent = traceText(current);
    if (focusTarget === "table") {
      body?.querySelector<HTMLElement>('[aria-selected="true"]')?.focus();
    }
    if (focusTarget === "map") {
      map?.querySelector<SVGElement>('[aria-pressed="true"]')?.focus();
    }
  };

  doc.querySelectorAll<HTMLInputElement>('input[name="metric"]').forEach((input) => {
    input.addEventListener("change", () => {
      if (input.checked && isMetric(input.value)) {
        metric = input.value;
        draw();
      }
    });
  });
  const jsonButton = doc.getElementById("dl-json");
  const mdButton = doc.getElementById("dl-md");
  jsonButton?.addEventListener("click", () => download("atlas.json", JSON.stringify(atlas, null, 2), "application/json"));
  mdButton?.addEventListener("click", () => download("summary.md", summary, "text/markdown"));
  draw();
}

function isMetric(value: string): value is MetricKey {
  return value === "installed_nameplate_mw" || value === "queued_requested_mw" || value === "headroom_mw";
}

function drawTable(
  doc: Document,
  body: HTMLElement | null,
  rows: ViewRow[],
  selected: string,
  onSelect: (regionId: string) => void,
): void {
  if (!body) return;
  body.replaceChildren();
  for (const row of rows) {
    const tr = doc.createElement("tr");
    tr.tabIndex = 0;
    tr.setAttribute("aria-selected", row.regionId === selected ? "true" : "false");
    const name = doc.createElement("th");
    name.scope = "row";
    name.textContent = row.name;
    const status = doc.createElement("td");
    const badge = doc.createElement("span");
    badge.className = `badge status-${row.displayStatus}`;
    badge.textContent = row.displayStatus;
    status.appendChild(badge);
    const value = doc.createElement("td");
    value.textContent = valueText(row);
    const date = doc.createElement("td");
    date.textContent = row.releaseDate || "—";
    const coverage = doc.createElement("td");
    coverage.textContent = row.coverage;
    tr.append(name, status, value, date, coverage);
    const select = () => onSelect(row.regionId);
    tr.addEventListener("click", select);
    tr.addEventListener("keydown", (event) => {
      if (event.key === "Enter" || event.key === " ") {
        event.preventDefault();
        select();
      }
    });
    body.appendChild(tr);
  }
}

function drawMap(
  doc: Document,
  host: HTMLElement | null,
  geo: Geo,
  rows: ViewRow[],
  selected: string,
  onSelect: (regionId: string) => void,
): void {
  if (!host) return;
  host.replaceChildren();
  const svg = doc.createElementNS("http://www.w3.org/2000/svg", "svg");
  svg.setAttribute("viewBox", "-0.2 -0.2 3.4 3.4");
  svg.setAttribute("role", "group");
  svg.setAttribute("aria-label", "Synthetic region map");
  const byID = new Map(rows.map((row) => [row.regionId, row]));
  for (const feature of geo.features) {
    const row = byID.get(feature.properties.region_id);
    if (!row) continue;
    const ring = feature.geometry.coordinates[0] ?? [];
    const d = ring.map((point, index) => {
      const x = point[0];
      const y = 3 - point[1];
      return `${index === 0 ? "M" : "L"}${x},${y}`;
    }).join(" ");
    const path = doc.createElementNS("http://www.w3.org/2000/svg", "path");
    path.setAttribute("d", `${d} Z`);
    path.setAttribute("tabindex", "0");
    path.setAttribute("role", "button");
    path.setAttribute("aria-pressed", row.regionId === selected ? "true" : "false");
    path.setAttribute("aria-label", `${row.name}, ${row.displayStatus}, ${valueText(row)}`);
    const fill = fillFor(row.displayStatus);
    path.setAttribute("fill", fill);
    path.setAttribute("stroke", row.regionId === selected ? "#0b3a4a" : "#1b1914");
    path.setAttribute("stroke-width", row.regionId === selected ? "0.08" : "0.03");
    path.setAttribute("stroke-dasharray", row.displayStatus === "unobserved" || row.displayStatus === "missing" || row.displayStatus === "unknown" ? "0.08 0.05" : "");
    const select = () => onSelect(row.regionId);
    path.addEventListener("click", select);
    path.addEventListener("keydown", (event) => {
      if (event.key === "Enter" || event.key === " ") {
        event.preventDefault();
        select();
      }
    });
    svg.appendChild(path);
    const label = doc.createElementNS("http://www.w3.org/2000/svg", "text");
    const xs = ring.map((point) => point[0]);
    const ys = ring.map((point) => 3 - point[1]);
    label.setAttribute("x", String(average(xs)));
    label.setAttribute("y", String(average(ys)));
    label.setAttribute("text-anchor", "middle");
    label.setAttribute("font-size", "0.16");
    label.textContent = `${row.regionId} ${row.displayStatus}`;
    svg.appendChild(label);
  }
  host.appendChild(svg);
}

function fillFor(status: string): string {
  switch (status) {
    case "available": return "#d7efe6";
    case "stale": return "#f8e7c4";
    case "incompatible": return "#f8d6d6";
    case "unobserved": return "#e4ebf5";
    default: return "#e6e8eb";
  }
}

function average(values: number[]): number {
  if (values.length === 0) return 0;
  return values.reduce((sum, value) => sum + value, 0) / values.length;
}

function download(filename: string, text: string, type: string): void {
  const blob = new Blob([text], { type });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  link.click();
  URL.revokeObjectURL(url);
}

function boot(): void {
  if (typeof document === "undefined") return;
  const atlasNode = document.getElementById("gridatlas-atlas");
  const geoNode = document.getElementById("gridatlas-geo");
  const summaryNode = document.getElementById("gridatlas-summary");
  if (!atlasNode?.textContent || !geoNode?.textContent || !summaryNode?.textContent) return;
  const atlas = JSON.parse(atlasNode.textContent) as Atlas;
  const geo = JSON.parse(geoNode.textContent) as Geo;
  const summary = JSON.parse(summaryNode.textContent) as string;
  mount(document, atlas, geo, summary);
}

boot();
