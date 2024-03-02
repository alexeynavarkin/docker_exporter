.PHONY: default publish-image

default:
	@echo "no default target"

publish-image:
	docker buildx build -f deployments/Dockerfile -t alexnav/docker-exporter:0.0.7 --platform=linux/amd64,linux/arm64 --push .
