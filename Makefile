APP_NAME    := fungus-wars
VERSION     ?= $(shell git describe --tags --always --dirty)
DATE        ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
LDFLAGS     := -s -w -X main.Version=$(VERSION) -X main.BuildDate=$(DATE)
GOFLAGS     := -trimpath -ldflags "$(LDFLAGS)"
BIN_DIR     ?= .
BIN         := $(BIN_DIR)/fungus-wars
BUILD_DIR   := build

PKG_INCLUDE          := fungus-wars static
CONTAINER_ENGINE     ?= podman
CONTAINERS_CONF      ?= $(CURDIR)/pkg/containers.conf
OCI_IMAGE_TAG        ?= latest
OCI_IMAGE            := $(APP_NAME):$(OCI_IMAGE_TAG)
RPM_VERSION          := $(shell echo "$(VERSION)" | perl -nE 'if(/^v(\d+\.\d+\.\d+)(?:-(dirty))?$$/){say "$$1$$2"}else{s/[^0-9a-z]//gi; say "0.untagged.$$_"}')
RPM_SPEC             := pkg/rpm.spec
RPM_TARBALL          := $(BUILD_DIR)/$(APP_NAME)-$(RPM_VERSION).rpmsource.tar.gz
RPM_BUILD_IMAGE      ?= docker.io/library/rockylinux:9
NO_CONTAINER_CLEANUP ?=

GO_FILES            := $(wildcard *.go) $(wildcard internal/*/*.go)
GO_GENERATE_SOURCES := how_to_play.md
DOCROOT_FILES       := $(wildcard static/* static/*/*)

# ------------------------------------------------------------
# targets related to development and building the application
# ------------------------------------------------------------

fungus-wars: $(GO_FILES) $(GO_GENERATE_SOURCES)
	go generate
	mkdir -p "$(BIN_DIR)"
	CGO_ENABLED=0 go build $(GOFLAGS) -o "$(BIN)"

.PHONY: build
build: fungus-wars

.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)

.PHONY: fmt
fmt:
	find . -name "*.go" -exec go fmt {} \;

# Not necessary for most dev work. For skipping pkg-* steps but running as a service.
.PHONY: install
install: build $(DOCROOT_FILES)
	mkdir -p /usr/share/fungus-wars/
	cp -av static /usr/share/fungus-wars/
	cp -vf fungus-wars /usr/local/bin/
	which setcap >/dev/null && setcap cap_net_bind_service=+ep /usr/local/bin/fungus-wars || :
	test -d /etc/systemd/system && cp -v pkg/fungus-wars.service /etc/systemd/system/
	which systemctl >/dev/null && systemctl daemon-reload
	getent group fungus-wars >/dev/null || groupadd --system fungus-wars
	getent passwd fungus-wars >/dev/null || useradd --system -g fungus-wars fungus-wars
	which systemctl >/dev/null && systemctl restart fungus-wars.service

.PHONY: uninstall
uninstall:
	which systemctl >/dev/null && systemctl disable --now fungus-wars.service || :
	rm -rfv /usr/share/fungus-wars/
	rm -rfv /usr/local/bin/fungus-wars
	rm -rfv /etc/systemd/system/fungus-wars.service
	getent passwd fungus-wars >/dev/null && userdel fungus-wars || :
	getent group fungus-wars >/dev/null && groupdel fungus-wars || :

# ----------------------------------------------------------
# pkg-* targets build the app into a distributable artifact
# ----------------------------------------------------------

# tar.gz

.PHONY: pkg-tar.gz
pkg-tar.gz: $(BUILD_DIR)/$(APP_NAME)-$(VERSION).tar.gz

$(BUILD_DIR)/$(APP_NAME)-$(VERSION).tar.gz: build $(DOCROOT_FILES)
	mkdir -p $(BUILD_DIR)
	tar -czf $@ $(PKG_INCLUDE)


# runnable container

.PHONY: pkg-oci-image pkg-docker pkg-podman
pkg-oci-image:
	CONTAINERS_CONF=$(CONTAINERS_CONF) \
	$(CONTAINER_ENGINE) build \
		-f pkg/Containerfile \
		-t $(OCI_IMAGE) \
		--build-arg VERSION="$(VERSION)" \
		--build-arg DATE="$(DATE)" \
		.

pkg-docker:
	$(MAKE) pkg-oci-image CONTAINER_ENGINE=docker

pkg-podman:
	$(MAKE) pkg-oci-image CONTAINER_ENGINE=podman

# RPM

.PHONY: pkg-rpm-tar.gz
pkg-rpm-tar.gz: $(RPM_TARBALL)

$(RPM_TARBALL): $(GO_FILES) $(GO_GENERATE_SOURCES) $(DOCROOT_FILES) $(RPM_SPEC)
	mkdir -p $(BUILD_DIR)
	# Ensure the tarball contains a top-level dir matching %{name}-%{version}
	git archive \
		--format=tar.gz \
		--prefix=$(APP_NAME)-$(RPM_VERSION)/ \
		-o $@ \
		HEAD

.PHONY: pkg-rpm
pkg-rpm: pkg-rpm-tar.gz
	mkdir -p "$(BUILD_DIR)"
	$(CONTAINER_ENGINE) run $(if $(NO_CONTAINER_CLEANUP),,--rm) \
		-v "$(CURDIR)":/src:Z \
		-v "$(CURDIR)/$(BUILD_DIR)":/out:Z \
		-w /src \
		$(RPM_BUILD_IMAGE) \
		bash -lc '\
			set -euo pipefail; \
			# Setup \
			dnf -y install rpm-build rpmdevtools make golang git systemd-rpm-macros shadow-utils libcap && \
			rpmdev-setuptree && \
			cp -v /src/$(RPM_TARBALL) ~/rpmbuild/SOURCES/ && \
			cp -v /src/$(RPM_SPEC) ~/rpmbuild/SPECS/ && \
			# Build job variables \
			GIT="git -C /src -c safe.directory=/src"; \
			export SOURCE_DATE_EPOCH=`$$GIT log -1 --pretty=%ct`; \
			RPM_GIT_COMMIT=`$$GIT rev-parse HEAD`; \
			RPM_GIT_STATUS=`$$GIT status --porcelain | grep -q . && echo dirty || echo clean`; \
			rpmbuild -ba ~/rpmbuild/SPECS/$(notdir $(RPM_SPEC)) \
				--define "_app_rpm_version $(RPM_VERSION)" \
				--define "_app_version $(VERSION)" \
				--define "_app_git_commit $$RPM_GIT_COMMIT" \
				--define "_app_git_status $$RPM_GIT_STATUS" && \
			# Copy out of the container \
			install -m 0644 -v ~/rpmbuild/RPMS/*/*.rpm /out/ && \
			install -m 0644 -v ~/rpmbuild/SRPMS/*.src.rpm /out/; \
		'

