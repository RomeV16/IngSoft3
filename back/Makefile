SHELL := /bin/bash

.PHONY: test cover check-cover cobertura

test:
	go test ./...

cover:
	go test ./... -covermode=count -coverprofile=cover.out

check-cover: cover
	@PCT=$$(go tool cover -func=cover.out | awk '/total:/ {gsub(/%/,"",$$3); print $$3}'); \
	THRESH=70; \
	echo "Total coverage: $$PCT% (threshold $$THRESH%)"; \
	awk -v p="$$PCT" -v t="$$THRESH" 'BEGIN{exit (p+0>=t)?0:1}' || (echo "Coverage below threshold" && exit 1)

cobertura: cover
	@which gocover-cobertura >/dev/null 2>&1 || { echo "Installing gocover-cobertura"; go install github.com/t-yuki/gocover-cobertura@latest; } ;
	gocover-cobertura < cover.out > cobertura.xml
	echo "Generated cobertura.xml"


