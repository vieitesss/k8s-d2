_default:
	just --list

snapshot:
	#!/usr/bin/env bash
	name=$(git rev-parse --short HEAD)
	git tag -f "$name"
	git push -f origin "$name"

tag version:
	git tag v{{version}} && git push origin v{{version}}
