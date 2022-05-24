#!/usr/bin/env bash
cd ..
CGO_ENABLED=1 go test -v
CGO_ENABLED=1 go test -bench=.