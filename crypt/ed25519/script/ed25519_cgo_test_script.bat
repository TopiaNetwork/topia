cd ..
go mod tidy
go env -w CGO_ENABLED=1
go test -v
go test -bench="."