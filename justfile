outputFile := "cluster.d2"
outputImage := "cluster.svg"

alias b := build
alias g := generate
alias tl := test_local


_default:
	just --list

snapshot:
	#!/usr/bin/env bash
	name=$(git rev-parse --short HEAD)
	git tag -f "$name"
	git push -f origin "$name"

tag version:
	git tag {{version}} && git push origin {{version}}

build:
	go build -o k8sdd

run *parameters: build
	./k8sdd {{parameters}}

generate *parameters:
	just run {{parameters}} -o {{outputFile}}
	d2 {{outputFile}} {{outputImage}}
	open {{outputImage}}

[working-directory: "dagger"]
test_local:
	dagger call run --docker-socket /var/run/docker.sock --kind-svc tcp://localhost:61254 --kubeconfig file://~/.kube
