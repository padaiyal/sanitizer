module github.com/padaiyal/sanitizer

go 1.22.5

require (
	github.com/PaesslerAG/jsonpath v0.1.2-0.20240529151134-87f681734c9c
	github.com/hexops/gotextdiff v1.0.3
	github.com/sergi/go-diff v1.3.1
	github.com/stretchr/testify v1.9.0
	github.com/tebeka/selenium v0.9.9
	github.com/tidwall/sjson v1.2.5
	gopkg.in/yaml.v3 v3.0.1
)
// To find which packages are using any of the indirect imports use `go mod why -m <indirect_imported_package>` Ex. go mod why -m github.com/blang/semver
require (
	github.com/PaesslerAG/gval v1.2.2 // indirect
	github.com/blang/semver v3.5.1+incompatible // indirect // Used by selenium
	github.com/davecgh/go-spew v1.1.1 // indirect // used by testify
	github.com/pmezard/go-difflib v1.0.0 // indirect // used by testify
	github.com/shopspring/decimal v1.3.1 // indirect
	github.com/tidwall/gjson v1.14.2 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
)
