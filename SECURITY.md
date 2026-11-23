# Security

GridAtlas reads local CSV and GeoJSON and writes a report directory. It does not authenticate users and it does not call a network service.

## Reporting a vulnerability

Open a private security advisory on this GitHub repository. Please include the version, the command you ran, and a fixture that shows the problem. Do not include credentials or personal data in the report.

## Supported versions

| Version | Supported |
| --- | --- |
| 0.1.x | yes |

## What is in scope

- Reading past the input directory through a snapshot path.
- Writing outside the chosen output directory.
- Treating a malformed file as a successful coverage score.
- HTML or Markdown injection from a source string in the report.

## What is out of scope

The accuracy of public grid datasets, estimates of electrical headroom, and hosted map or data APIs. This program does not compute available capacity.
