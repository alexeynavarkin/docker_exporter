.PHONY: default push-image

default:
	@echo "no default target"

push-image:
	docker buildx build -f deployments/Dockerfile -t alexnav/docker-exporter:0.0.2 --platform=linux/amd64,linux/arm64 --push .
