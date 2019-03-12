# Stash

A Go package for disk-based blob cache.

## Installation

Install Stash using the go get command:

    $ go get github.com/chikamim/stash

The only dependency is the Go distribution.

## Motivation

This package allows us to reduce calls to our blob storage by caching the most recently used blobs to disk.

## Documentation

- [Reference](https://godoc.org/github.com/chikamim/stash)

## Contributing

Contributions are welcome.

## License

Stash is available under the [BSD (3-Clause) License](https://opensource.org/licenses/BSD-3-Clause).

## Disclaimer

The package is a work in progress. It is functional, but does not claim to be production-ready. Please use it at your own risk.
