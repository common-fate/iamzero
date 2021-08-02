module github.com/common-fate/iamzero

go 1.16

require (
	github.com/aws/aws-sdk-go-v2 v1.7.0
	github.com/aws/aws-sdk-go-v2/config v1.4.1
	github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue v1.1.2
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.4.0
	github.com/aws/aws-sdk-go-v2/service/iam v1.3.0
	github.com/aws/aws-sdk-go-v2/service/kms v1.2.2
	github.com/go-chi/chi v1.5.4
	github.com/google/uuid v1.2.0
	github.com/peterbourgon/ff/v3 v3.0.0
	github.com/pkg/errors v0.9.1
	github.com/smartystreets/goconvey v1.6.4 // indirect
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.21.0
	go.opentelemetry.io/otel v1.0.0-RC1
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.0.0-RC1
	go.opentelemetry.io/otel/sdk v1.0.0-RC1
	go.opentelemetry.io/otel/trace v1.0.0-RC1
	go.uber.org/zap v1.16.0
	golang.org/x/lint v0.0.0-20200302205851-738671d3881b // indirect
	golang.org/x/tools v0.0.0-20200825202427-b303f430e36d // indirect
	google.golang.org/grpc v1.38.0
	gopkg.in/ini.v1 v1.62.0
	honnef.co/go/tools v0.0.1-2020.1.4 // indirect
)
