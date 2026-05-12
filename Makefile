SHELL := /bin/bash

.PHONY: proto proto-generate

proto: proto-generate

proto-generate:
	@bash scripts/gen-proto.sh
