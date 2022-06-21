#!/usr/bin/env bash
cd ..
CGO_ENABLED=0 go test -v
CGO_ENABLED=0 go test -bench=.