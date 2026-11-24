module linear-dashboard/backend

go 1.26.0

require (
	github.com/go-kratos/kratos/v3 v3.0.0
	github.com/golang-migrate/migrate/v4 v4.20.1
	github.com/jackc/pgx/v5 v5.13.0
	github.com/robfig/cron/v3 v3.0.1
	golang.org/x/sync v0.22.0
	google.golang.org/genproto/googleapis/api v0.0.0-20261116203732-fd2c7689a808
	google.golang.org/protobuf v1.36.12
)

require (
	github.com/cockroachdb/cockroach-go/v2 v2.1.1 // indirect
	github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v3 v3.0.0-20260626125723-668db92c2c00 // indirect
	github.com/go-playground/form/v4 v4.3.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.3 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/lib/pq v1.10.9 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20261110000943-09e21b237f2c // indirect
	google.golang.org/grpc v1.83.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

tool (
	github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v3
	google.golang.org/protobuf/cmd/protoc-gen-go
)
