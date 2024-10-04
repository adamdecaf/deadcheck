# deadcheck

[![GoDoc](https://godoc.org/github.com/adamdecaf/csvq?status.svg)](https://godoc.org/github.com/adamdecaf/csvq)
[![Build Status](https://github.com/adamdecaf/csvq/workflows/Go/badge.svg)](https://github.com/adamdecaf/csvq/actions)
[![Coverage Status](https://codecov.io/gh/adamdecaf/csvq/branch/master/graph/badge.svg)](https://codecov.io/gh/adamdecaf/csvq)
[![Go Report Card](https://goreportcard.com/badge/github.com/adamdecaf/csvq)](https://goreportcard.com/report/github.com/adamdecaf/csvq)
[![Apache 2 License](https://img.shields.io/badge/license-Apache2-blue.svg)](https://raw.githubusercontent.com/adamdecaf/csvq/master/LICENSE)

deadcheck is an Operator Presence Control ("OPC") system to alert when an action has not occurred within a predefined window. This is also called a [Dead man's switch](https://en.wikipedia.org/wiki/Dead_man's_switch). deadcheck relies on setting up a schedule with a third-party service and suppressing notifications. That way a failure to check-in with deadcheck or a failure within deadcheck will cause a notification to be triggered. High quality services are chosen as integrations for deadcheck as failures to fire are a known short coming of deadcheck.

## Install

Download the [latest release for your architecture](https://github.com/adamdecaf/deadcheck/releases/latest).

## Configuration
```yaml
checks:
  - id: "hourly-sync"
    name: "Upload data every hour"
    description: "<string>"
    schedule:
      every: "1h"
    alert:
      pagerduty:
        apiKey: "<string>"
        escalationPolicy: "<string>"

  - id: "2pm-checkin"
    name: "Reports Finalized"
    schedule:
      weekdays:
        timezone: "America/New_York"
        times:
          - at: "14:00"
            tolerance: "5m"
    alert:
      pagerduty:
        apiKey: "<string>"
        escalationPolicy: "<string>"

  - id: "5pm-close"
    name: "Close out for the day"
    schedule:
      bankingDays:
        timezone: "America/New_York"
        times:
          - at: "17:00"
            tolerance: "5m"
    alert:
      pagerduty:
        apiKey: "<string>"
        escalationPolicy: "<string>"
```


## Usage

`PUT /v1/checks/{id}/check-in`

Successful response, or failure. Optional: `Extension` time.Duration value


## Integrations

- PagerDuty: A service is used and incident created but there is a maintenance window preventing notifications. Each successful trigger pushes the maintenance window out longer into the future.

## Supported and tested platforms

- 64-bit Linux (Ubuntu, Debian), macOS, and Windows

## License

Apache License 2.0 - See [LICENSE](LICENSE) for details.
