# -wvh- go project utility makefile
#

ROOTDIR := ${CURDIR}
BINDIR := $(ROOTDIR)/bin
WEBPORT := 8080

DOCKER_LOGIN := quay.io
DOCKER_REPO := quay.io/wvh/urn-web

ENVFILE=docker/.env
DB_CONTAINER=urndb
NETWORK_NAME=urnnet
VERSIONFILE=internal/version

# enforce module mode
export GO111MODULE=on


### VCS
TAG = $(shell git describe --tags --always --dirty="-dev" 2>/dev/null)
HASH = $(shell git rev-parse --short HEAD 2>/dev/null)
BRANCH = $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null)
REPO = $(shell git ls-remote --get-url 2>/dev/null)
REPOLINK = $(shell test -n "$(REPO)" && test -x $(SOURCELINK) && ${GOBIN}/sourcelink $(REPO) $(HASH) $(BRANCH) 2>/dev/null || echo)
VERSION_PACKAGE = $(shell go list -f '{{.ImportPath}}' ./$(VERSIONFILE))

LDFLAGS = -s -w

### link in version info
ifneq ($(wildcard .git/.),)
HAVE_GIT := true
LDFLAGS += -X $(VERSION_PACKAGE).Hash=$(HASH) -X $(VERSION_PACKAGE).Tag=$(TAG) -X $(VERSION_PACKAGE).Branch=$(BRANCH) -X $(VERSION_PACKAGE).Repo=$(REPOLINK)
#$(info found git)
else
HAVE_GIT := false
#$(info did not find git)
endif

### link in version info
#ifdef $(HASH)
#	LDFLAGS += "-s -w -X $(VERSION_PACKAGE).Hash=$(HASH) -X $(VERSION_PACKAGE).Tag=$(TAG) -X $(VERSION_PACKAGE).Branch=$(BRANCH) -X $(VERSION_PACKAGE).Repo=$(REPOLINK)"
#endif

### trim file system paths from executables
# ... for go < 1.10
#TRIMFLAGS := -gcflags=-trimpath=$(PARENTDIR) -asmflags=-trimpath=$(PARENTDIR)
# ... for go >= 1.10
#TRIMFLAGS := -gcflags=all=-trimpath=$(PARENTDIR) -asmflags=all=-trimpath=$(PARENTDIR)
# ... for go >= 1.13
TRIMFLAGS := -trimpath

.PHONY: all
all: | $(BINDIR)
	go build -v -ldflags "$(LDFLAGS)" $(TRIMFLAGS) -o $(BINDIR) ./cmd/...

.PHONY: web
web: | $(BINDIR)
	go build -v -ldflags "$(LDFLAGS)" $(TRIMFLAGS) -o $(BINDIR)/ ./cmd/web

.PHONY: vars
vars:
	export TAG=$(TAG) HASH=$(HASH) BRANCH=$(BRANCH) REPO=$(REPO) REPOLINK=$(REPOLINK) TESTVAR=foo; \
	echo TAG=${TAG} $$TAG TESTVAR=${TESTVAR} $$TESTVAR

$(BINDIR):
	mkdir -p $(BINDIR)

.PHONY: test
test:
	go test -v ./...

.PHONY: clean
clean:
	rm -f $(BINDIR)/*
	go clean ./cmd/...

.PHONY: gofmt
gofmt:
	gofmt -s -w .

.PHONY: serve
#serve: PGHOST := $(or $(PGHOST),$(shell docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $(DB_CONTAINER)))
serve: web
	@if [ -n "$(PGHOST)" ]; then \
		echo make: using $(PGHOST) for database connections; \
	else \
		echo warning: "PGHOST is unset"; \
	fi
	PGHOST=$(PGHOST) sh -ac '. $(ENVFILE) && $(BINDIR)/web'

.PHONY: docker-build
docker-build:
	# docker can add multiple tags; :latest is the default when no tag is given
	docker build -t web -t web:$(TAG) -f docker/web/Dockerfile.cache .

.PHONY: docker-tag
docker-tag: docker-build
	docker tag web $(DOCKER_REPO):$(TAG)

.PHONY: docker-push
docker-push: docker-tag
	docker push $(DOCKER_REPO)

.PHONY: docker-serve
docker-serve: docker-build
	docker run -it -p $(WEBPORT):8080 --rm web

.PHONY: docker-cmd
docker-cmd: docker-build
	docker run -it --rm --entrypoint /bin/hello web

.PHONY: dev-up
dev-up:
	cd docker && TAG=$(TAG) docker-compose up

.PHONY: dev-down
dev-down:
	cd docker && docker-compose down

.PHONY: dev-list
dev-list:
	docker network inspect -f '{{range .Containers}}{{println .Name .IPv4Address}}{{end}}' $(NETWORK_NAME)

.PHONY: help
help:
	@echo 'targets defined in this makefile:'
	@echo
	@echo 'LOCAL COMMANDS'
	@echo '  make all            build all go commands ($(BINDIR))'
	@echo '  make web            build the web server only'
	@echo '  make serve          run the web server locally'
	@echo '  make test           run tests'
	@echo '  make clean          clean up build artifacts'
	@echo
	@echo 'DOCKER'
	@echo '  make docker-build   build project inside container'
	@echo '  make docker-serve   serve requests on port $(WEBPORT)'
	@echo '  make docker-push    push image to remote repo $(DOCKER_REPO)'
	@echo '  make docker-cmd     run one command'
	@echo
	@echo 'DOCKER-COMPOSE'
	@echo '  make dev-up         start project in container environment'
	@echo '  make dev-list       list running containers and their IP addresses'
	@echo '  make dev-down       stop container environment'
	@echo
