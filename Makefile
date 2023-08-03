DOCKER_IMAGE       ?= luyuanxin1995/bricksllm
VERSION   := $(shell git describe --tags || echo "v0.0.0")
VER_CUT   := $(shell echo $(VERSION) | cut -c2-)

docker:
	@docker build -f ./build/Dockerfile . -t $(DOCKER_IMAGE):$(VER_CUT)
	@docker tag $(DOCKER_IMAGE):$(VER_CUT) $(DOCKER_IMAGE):latest