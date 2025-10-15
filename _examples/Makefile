EXAMPLE := playground
EXAMPLES = cli

ifeq (,$(filter $(EXAMPLE),$(EXAMPLES) playground))
$(error example '$(EXAMPLE)' not found in _examples)
endif
ifeq ($(EXAMPLE),playground)
ifeq ($(wildcard playground/),)
$(error directory 'playground' not found)
endif
endif

ifeq ($(wildcard $(EXAMPLE)/.env),)
$(error file '$(EXAMPLE)/.env' not found)
endif

include ./$(EXAMPLE)/.env
export

.PHONY: cmd
cmd:
	@go build -o ../bin/$(EXAMPLE) ./$(EXAMPLE)/main.go
	@../bin/$(EXAMPLE)
