# Luxon

[![MIT License][license-image]][license] [![Build Status][github-action-image]][github-action-url] [![PRs welcome][contributing-image]][contributing-url]

Luxon is a library for working with dates and times in Go. It is a port of Luxon 3.7.2, the
JavaScript library: the API is Luxon's with Go names, and every result, time zone data and
localized strings included, is what the JavaScript library gives on Node.js 24.22.1 (ICU 78.3,
CLDR 48, tz 2026e).

```go
luxon.DateTimeNow().SetZone("America/New_York").Minus(luxon.Obj{"weeks", 1.0}).EndOf("day").ToISO()
```

## 🚀 The road to Luxon 4.0

Please [read about and share your feedback](https://github.com/moment/luxon/discussions/1742) on the plans for the next major version of Luxon!

## Upgrading to 3.0

[Guide](https://moment.github.io/luxon/#upgrading)

## Features
 * DateTime, Duration, and Interval types.
 * Immutable, chainable, unambiguous API.
 * Parsing and formatting for common and custom formats.
 * Time zone, locale, calendar and numbering-system data built in; no dependencies.

## Install

The module path is `github.com/moment/luxon-go` (Go 1.27, standard library only):

```go
import luxon "github.com/moment/luxon-go"
```

## Documentation

The documentation describes the JavaScript API, which maps to Go mechanically: `DateTime.fromISO(...)`
is `luxon.DateTimeFromISO(...)`, `dt.toISO()` is `dt.ToISO()`, `Info.months(...)` is
`luxon.Info.Months(...)`, and an object literal is a `luxon.Obj` of alternating keys and values.

* [General documentation](https://moment.github.io/luxon/#/?id=luxon)
* [API docs](https://moment.github.io/luxon/api-docs/index.html)
* [Quick tour](https://moment.github.io/luxon/#/tour)
* [For Moment users](https://moment.github.io/luxon/#/moment)
* [Why does Luxon exist?](https://moment.github.io/luxon/#/why)
* [A quick demo](https://moment.github.io/luxon/demo/global.html)

## Development

```sh
go build ./... && go vet ./... && go test ./...
```

See [contributing](CONTRIBUTING.md).

![Phasers to stun][phasers-image]

[license-image]: https://img.shields.io/badge/license-MIT-blue.svg
[license]: LICENSE.md

[github-action-image]: https://github.com/moment/luxon/actions/workflows/test.yml/badge.svg
[github-action-url]: https://github.com/moment/luxon/actions/workflows/test.yml

[contributing-url]: https://github.com/moment/luxon/blob/master/CONTRIBUTING.md
[contributing-image]: https://img.shields.io/badge/PRs-welcome-brightgreen.svg

[phasers-image]: https://img.shields.io/badge/phasers-stun-brightgreen.svg
