# Overlap review

Access date: 2026-11-23.

Decision: **proceed** with a separate GridAtlas repository.

GridAtlas is a local inventory of whether coarse regional planning figures exist, what definition they use, and how fresh they are. The acceptance case is a frozen as-of date over a synthetic three-region fixture: installed nameplate and queued requested MW stay separate, repeated request IDs do not inflate the queue sum, a missing generator source stays missing rather than zero, a snapshot past its freshness threshold becomes stale without dropping its original date or value, and electrical headroom stays unobserved. None of the tools below accept that fixture or publish that coverage state.

## Shared fixture and metric

Held constant for every contender:

- Input: region `SYN-NORTH` with operating nameplate rows of 100 MW and 20 MW (120 MW), active queue requests `Q-20` = 20 MW (listed twice), `Q-30` = 30 MW, plus an ambiguous multi-technology request, a withdrawn request, and an omitted row. Region `SYN-SOUTH` has no generator source.
- Expected artifact: installed 120 MW, queued 50 MW, headroom unobserved, south installed missing (not 0).
- Acceptance metric: every region and metric matches that coverage state and aggregate, with dated provenance, and no synthesized headroom.

Observed local scoring of that fixture on 2026-11-23 (Python, standard library only; not a contender):

```text
gridstatus import: unavailable: ModuleNotFoundError No module named 'gridstatus'
deduped_queued_mw 50
ambiguous_requests 1
naive_headroom_mw 15
acceptance_headroom unobserved
naive_matches_acceptance False
```

A subtraction of queued MW from installed MW produces a headroom number. That fails the acceptance metric even before deduplication. `gridstatus` 0.37.0 is not installed here and its documented entry point fetches ISO websites rather than reading this fixture.

GitHub repository search `interconnection queue coverage atlas capacity` returned `total_count` 0 on 2026-11-23. npm search `gridatlas` returned no matches.

## Capability matrix

| Tool | Primary URL | Version observed | Code license | Dataset terms | Evidence | Already matches |
| --- | --- | --- | --- | --- | --- | --- |
| Berkeley Lab Queued Up | https://emp.lbl.gov/queues | 2026 edition, workbook updated May 2026, PDF June 2026, data through end of 2025 | No software repository. The June 2026 slide deck is a LBNL manuscript under DOE contract DE-AC02-05CH11231; the copyright notice grants the US government a publication license and does not state an OSI license for the workbook | Public page publishes an XLSX and PDF. The deck says some gaps are filled with non-public data from transmission providers. Redistribution of the workbook was not settled | Documentation | Active-queue GW rollups, technology mix, and a coverage map of included balancing areas. No installed-versus-queued definition check, no stale/missing/zero/incompatible cell, no unobserved-headroom field |
| EIA-860M | https://www.eia.gov/electricity/data/eia860m/ | September 2026 vintage, page release date October 23, 2026, next release November 24, 2026 | US government work | Public domain. Reuse is allowed with an acknowledgment that includes the publication date (https://www.eia.gov/about/copyrights_reuse.php, read 2026-11-23). EIA logo and some photographs are restricted | Documentation | Generator nameplate MW and a monthly vintage. No queue product, no freshness status against a caller-supplied as-of date, no planning-region crosswalk inside the file |
| gridstatus | https://github.com/gridstatus/gridstatus | PyPI 0.37.0; GitHub `gridstatus/gridstatus` pushed 2026-11-20; 456 stars | BSD-3-Clause | Fetches upstream ISO, EIA, and EPA files under those publishers' terms. The hosted Grid Status API is a separate product | Documentation, plus an observed failed import (`ModuleNotFoundError`) | Interconnection-queue fetchers and EIA access. Not an offline coverage atlas. The open-source library does not score this fixture |
| PUDL | https://github.com/catalyst-cooperative/pudl | Documentation set labeled PUDL 2026.11.0; GitHub `catalyst-cooperative/pudl` pushed 2026-11-22; 622 stars | MIT for software. Data and docs CC-BY-4.0 | Processed EIA-860 and EIA-860M under that attribution license. Raw EIA inputs remain public domain | Documentation | Analysis-ready generator tables. Not a queue-versus-installed coverage map, and not a stale/missing/incompatible scorer |
| GridCapacity API | https://gridcapacityapi.com/docs/capacity-model | Coverage page stamped 2026-11-22 21:57 UTC | Not an open-source local library (no repo licensed for this review) | Rights are enforced per source. This review did not call the API | Documentation | Separates installed, requested, and operator-published available figures, and can reject stale records. Asset-level, hosted, and it serves operator "available" room, which this MVP must not invent or map as headroom |
| UK Grid Atlas | https://github.com/jacobwright32/uk-grid-atlas | `main` README fetched 2026-11-23; commit SHA not resolved | Not recorded in this pass | Mixes OSM, Elexon, and other feeds | Documentation | Plant map and live output. Explicitly out of scope (facility sites, operational feed) |
| GB Grid Queue Map | https://github.com/REAILab-works/uk-grid-map | README cites a NESO TEC snapshot of 19 November 2026 | Not recorded in this pass | NESO TEC plus OSM and PyPSA-GB | Documentation | One queue snapshot on a substation map, with a connected-MW comparison. Facility and circuit geometry, one country, no coverage-state inventory |
| interconnection-queue-analysis | https://github.com/keanuhea/interconnection-queue-analysis | Pushed 2026-05-18; GitHub license field null | None stated | Uses Queued Up 2025 | Repository metadata only | Withdrawal and corridor analysis. Not a coverage atlas |

## Probe outcomes

### Queued Up active-status rollup

Evidence class: documentation. Unresolved as a reproduced number.

The 2026 edition states an active sample of 8,244 projects and about 2,061 GW, and an operational sample of 4,789 projects and about 567 GW, from 7 ISO/RTOs and 50 non-ISO balancing areas (~98% of installed US generating capacity). "Active" includes requests in study and requests with an executed agreement that are not yet in commercial operation. Suspended (668 projects, ~137 GW) is separate. The 2024 edition, through the end of 2023, reported 11,597 active projects and about 2,598 GW from 7 ISO/RTOs and 44 non-ISO areas. Those two published rollups are not the same universe, so a single active-MW total is not comparable across editions without a coverage note.

This session did not download the 14.85 MB workbook. The deck says non-public provider data fills some gaps, so a local sum would not be a clean public-data reproduction. Queued Up also has no entry point for the shared synthetic fixture.

### EIA-860M region crosswalk

Evidence class: documentation. Unresolved as a computed aggregate.

The inventory is a preliminary monthly estimate of generators of 1 MW or greater. Capacities are nameplate estimates, not capacity commitments, and later months can correct them without a per-value explanation. The public page does not provide a planning-region field that matches an ISO footprint. A state or balancing-authority sum is an extra crosswalk with overlaps. That is not a defensible aggregate to ship as a regional total. Provenance path for a later import is still viable: public-domain monthly files, cited with the vintage date, aggregated only after an explicit supported crosswalk. Those files stay out of this repository.

### gridstatus and PUDL on the shared fixture

Evidence class: documentation, plus the observed import failure for `gridstatus`.

Both are ingestion pipelines. Running them would fetch or unpack upstream tables, not score `SYN-NORTH` / `SYN-SOUTH`. The hosted Grid Status API can require an account and has paid plans, so it is out of scope for this project.

### GridCapacity API, UK Grid Atlas, GB Grid Queue Map

Evidence class: documentation. Live calls were not made.

GridCapacity is the closest product conceptually and the strongest reason to narrow the claim: it already refuses to subtract a queue from a total, and it tracks source freshness. It does that for operator assets in Europe and elsewhere, behind a service, and it publishes operator-defined available room. GridAtlas does not need to reproduce that service. The MVP is an offline, coarse, US-planning-region coverage sheet with synthetic fixtures and an explicit unknown for headroom.

## Features that already match

- Queued Up already cleans active versus withdrawn queue rows and publishes a dated national snapshot.
- EIA-860M already publishes dated nameplate MW.
- gridstatus and PUDL already move those public files into tables.
- GridCapacity already models distinct capacity types and staleness.

What they do not match is the joint coverage question: for one coarse region, is the installed product present, is the queue product present, are the units and the region join defensible, is the vintage inside a caller-supplied threshold, and is headroom simply unobserved?

## Remaining uncertainties

- Exact SPDX-style terms for redistributing the Queued Up workbook. Treat them as unclear. Do not commit the workbook.
- Whether a single ISO active-MW tab in that workbook can be reproduced from public queue files alone. Not tested.
- A defensible state-or-BA to planning-region crosswalk for real EIA-860M rows. Not tested, and not required for the synthetic MVP.
- GridCapacity API pricing and a machine-readable license. Not required, because the project will not call it.

## Decision

Proceed. Build the local Go scorer and the TypeScript map against synthetic fixtures. Do not integrate into gridstatus, PUDL, or Queued Up: their workflows do not meet the acceptance case, and bending this MVP into a fetcher or a facility map would cross the stated boundary. A later contributor can add an optional importer that fills the same CSV contract from a public-domain EIA vintage or a queue extract the user already has rights to. Until then, ship fixtures, the manifest, and import instructions only.
