# deadcheck

[![GoDoc](https://godoc.org/github.com/adamdecaf/deadcheck?status.svg)](https://godoc.org/github.com/adamdecaf/deadcheck)
[![Build Status](https://github.com/adamdecaf/deadcheck/workflows/Go/badge.svg)](https://github.com/adamdecaf/deadcheck/actions)
[![Coverage Status](https://codecov.io/gh/adamdecaf/deadcheck/branch/master/graph/badge.svg)](https://codecov.io/gh/adamdecaf/deadcheck)
[![Go Report Card](https://goreportcard.com/badge/github.com/adamdecaf/deadcheck)](https://goreportcard.com/report/github.com/adamdecaf/deadcheck)
[![Apache 2 License](https://img.shields.io/badge/license-Apache2-blue.svg)](https://raw.githubusercontent.com/adamdecaf/deadcheck/master/LICENSE)
[![Docker Pulls](https://img.shields.io/docker/pulls/adamdecaf/deadcheck)](https://hub.docker.com/r/adamdecaf/deadcheck)

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
      every:
        interval: "1h"
        start: "14:00"
        end: "18:00"
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

```
PUT /v1/checks/{id}/check-in
```
```json
{"nextExpectedCheckIn":"2024-10-09T21:05:00Z"}
```

Successful response, or failure in the response.

## Integrations

- PagerDuty: A service is used and incident created but snoozed preventing notifications. Each successful check-in pushes the snooze out into the future until the next expected check-in.

## Supported and tested platforms

- 64-bit Linux (Ubuntu, Debian), macOS, and Windows

## License

Apache License 2.0 - See [LICENSE](LICENSE) for details.
