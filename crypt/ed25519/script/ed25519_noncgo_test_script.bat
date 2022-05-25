cd ..
go mod tidy
go env -w CGO_ENABLED=0
go test -v
go test -bench="."
go env -w CGO_ENABLED=1