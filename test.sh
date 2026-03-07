#!/bin/sh

go test -v
go run . -zones zones -logLevel debug -listen '[::]:1053'
