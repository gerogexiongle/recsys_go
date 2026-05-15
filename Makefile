# Project root = directory containing this Makefile (works from any clone path).
PROJECT_ROOT := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))

.PHONY: tidy build recommend rank e2e e2e-full e2e-lb

tidy:
	cd $(PROJECT_ROOT) && go mod tidy

build: recommend rank

recommend:
	cd $(PROJECT_ROOT) && go build -o bin/recommend-api ./services/recommend

rank:
	cd $(PROJECT_ROOT) && go build -o bin/rank-api ./services/rank

e2e:
	bash $(PROJECT_ROOT)/scripts/e2e.sh

e2e-full:
	bash $(PROJECT_ROOT)/scripts/e2e_full_chain.sh

e2e-lb:
	bash $(PROJECT_ROOT)/scripts/e2e_lb_dup_endpoints.sh
