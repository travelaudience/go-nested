# Nested Services

[![Go Reference](https://pkg.go.dev/badge/github.com/travelaudience/go-nested.svg)](https://pkg.go.dev/github.com/travelaudience/go-nested)
[![CircleCI](https://circleci.com/gh/travelaudience/go-nested.svg?style=svg)](https://circleci.com/gh/travelaudience/go-nested)

**go-nested** provides a simple library to simplify implementing nested services.

A nested service is a service that runs in the background, independently of the main program, and exposes an API
for communication with the main program or other services.  It functions much like a microservice, except that it
runs on the same machine and is compiled into the same binary.

A typical example of a nested service would be a caching layer, where the cache needs to be refreshed at some
regular interval an some external source.  The nested service abstracts away the logic of maintaining the cache,
exposing only an API to read from it.

This library provides a simple mechanism for the main program to monitor the nested service and take appropriate
action when the nested service is in an error state.
