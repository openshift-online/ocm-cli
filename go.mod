module github.com/openshift-online/ocm-cli

go 1.25.0

require (
	cloud.google.com/go/iam v1.1.8
	cloud.google.com/go/storage v1.39.1
	github.com/AlecAivazis/survey/v2 v2.3.7
	github.com/MicahParks/jwkset v0.7.0
	github.com/golang-jwt/jwt/v4 v4.5.2
	github.com/golang/glog v1.2.5
	github.com/googleapis/gax-go/v2 v2.14.1
	github.com/hashicorp/go-version v1.7.0
	github.com/m1/go-generate-password v0.2.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/nwidger/jsoncolor v0.3.2
	github.com/onsi/ginkgo/v2 v2.27.2
	github.com/onsi/gomega v1.38.2
	github.com/openshift-online/ocm-common v0.0.34
	github.com/openshift-online/ocm-sdk-go v0.1.486
	github.com/openshift/rosa v1.2.24
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.9.1
	github.com/spf13/pflag v1.0.9
	gitlab.com/c0b/go-ordered-json v0.0.0-20201030195603-febf46534d5a
	go.uber.org/mock v0.5.0
	golang.org/x/term v0.37.0
	golang.org/x/text v0.31.0
	google.golang.org/api v0.211.0
	google.golang.org/grpc v1.67.1
	gopkg.in/yaml.v3 v3.0.1
	k8s.io/apimachinery v0.35.0
)

require (
	cloud.google.com/go v0.112.2 // indirect
	cloud.google.com/go/auth v0.12.1 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.6 // indirect
	cloud.google.com/go/compute/metadata v0.5.2 // indirect
	github.com/99designs/go-keychain v0.0.0-20191008050251-8e49817e8af4 // indirect
	github.com/99designs/keyring v1.2.2 // indirect
	github.com/Masterminds/semver/v3 v3.4.0 // indirect
	github.com/alessio/shellescape v1.4.1 // indirect
	github.com/aws/aws-sdk-go v1.44.110 // indirect
	github.com/aymerick/douceur v0.2.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/briandowns/spinner v1.19.0 // indirect
	github.com/cenkalti/backoff/v4 v4.1.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/danieljoos/wincred v1.2.0 // indirect
	github.com/dvsekhvalnov/jose2go v1.6.0 // indirect
	github.com/evanphx/json-patch/v5 v5.6.0 // indirect
	github.com/fatih/color v1.13.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/godbus/dbus v0.0.0-20190726142602-4481cbc300e2 // indirect
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/pprof v0.0.0-20250403155104-27863c87afa6 // indirect
	github.com/google/s2a-go v0.1.8 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.4 // indirect
	github.com/gorilla/css v1.0.0 // indirect
	github.com/gsterjov/go-libsecret v0.0.0-20161001094733-a6f4afe4910c // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/itchyny/gojq v0.12.9 // indirect
	github.com/itchyny/timefmt-go v0.1.4 // indirect
	github.com/jackc/chunkreader/v2 v2.0.1 // indirect
	github.com/jackc/pgconn v1.14.3 // indirect
	github.com/jackc/pgio v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgproto3/v2 v2.3.3 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgtype v1.14.0 // indirect
	github.com/jackc/pgx/v4 v4.18.3 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.16 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/microcosm-cc/bluemonday v1.0.23 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/mtibben/percent v0.2.1 // indirect
	github.com/openshift-online/ocm-api-model/clientapi v0.0.440 // indirect
	github.com/openshift-online/ocm-api-model/model v0.0.440 // indirect
	github.com/prometheus/client_golang v1.13.0 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.37.0 // indirect
	github.com/prometheus/procfs v0.8.0 // indirect
	github.com/robfig/cron/v3 v3.0.1 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/skratchdot/open-golang v0.0.0-20200116055534-eef842397966 // indirect
	github.com/zalando/go-keyring v0.2.3 // indirect
	github.com/zgalor/weberr v0.7.0 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.54.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.54.0 // indirect
	go.opentelemetry.io/otel v1.29.0 // indirect
	go.opentelemetry.io/otel/metric v1.29.0 // indirect
	go.opentelemetry.io/otel/trace v1.29.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.45.0 // indirect
	golang.org/x/mod v0.29.0 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/oauth2 v0.24.0 // indirect
	golang.org/x/sync v0.18.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/time v0.9.0 // indirect
	golang.org/x/tools v0.38.0 // indirect
	google.golang.org/genproto v0.0.0-20240401170217-c3f982113cda // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20241209162323-e6fa225c2576 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241209162323-e6fa225c2576 // indirect
	google.golang.org/protobuf v1.36.8 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)
