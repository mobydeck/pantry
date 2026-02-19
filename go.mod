module pantry

go 1.25

require (
	github.com/asg017/sqlite-vec-go-bindings v0.1.6
	github.com/google/uuid v1.6.0
	github.com/modelcontextprotocol/go-sdk v1.3.1
	github.com/ncruces/go-sqlite3/gormlite v0.20.3
	github.com/spf13/cobra v1.10.2
	gopkg.in/yaml.v3 v3.0.1
	gorm.io/gorm v1.31.1
)

require (
	github.com/google/jsonschema-go v0.4.2 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/ncruces/go-sqlite3 v0.20.3 // indirect
	github.com/ncruces/julianday v1.0.0 // indirect
	github.com/segmentio/asm v1.2.1 // indirect
	github.com/segmentio/encoding v0.5.3 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/tetratelabs/wazero v1.11.0 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	golang.org/x/oauth2 v0.35.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.34.0 // indirect
)

// ncruces v0.20.3 required for sqlite-vec-go-bindings compatibility (WASM atomic ops)
