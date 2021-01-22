VERSION := $(shell git describe)
LDFLAGS := -ldflags "-X 'github.com/formicidae-tracker/zeus.ZEUS_VERSION=$(VERSION)'"

all: lib zeus/zeus arke-change-device-id arkedump arke-zeus-config zeus-calibrator zeus-cli/zeus-cli

lib:
	go test

clean:
	rm -f zeus/zeus
	rm -f tools/arke-change-device-id/arke-change-device-id
	rm -f tools/arkedump/arkedump
	rm -f tools/arke-zeus-config/arke-zeus-config
	rm -f tools/zeus-calibrator/zeus-calibrator
	rm -f zeus-cli/zeus-cli

zeus/zeus:
	cd zeus && go build $(LDFLAGS) && go test

arke-change-device-id:
	cd tools/arke-change-device-id && go build $(LDFLAGS)

arkedump:
	cd tools/arkedump && go build $(LDFLAGS)

arke-zeus-config:
	cd tools/arke-zeus-config && go build $(LDFLAGS)

zeus-calibrator:
	cd tools/zeus-calibrator && go build $(LDFLAGS)

zeus-cli/zeus-cli:
	cd zeus-cli && go build $(LDFLAGS)

INSTALL_PREFIX=/usr/local

install: all
	mkdir -p $(DESTDIR)$(INSTALL_PREFIX)/bin
	install zeus/zeus $(DESTDIR)$(INSTALL_PREFIX)/bin
	install tools/arke-change-device-id/arke-change-device-id $(DESTDIR)$(INSTALL_PREFIX)/bin
	install tools/arkedump/arkedump $(DESTDIR)$(INSTALL_PREFIX)/bin
	install tools/arke-zeus-config/arke-zeus-config $(DESTDIR)$(INSTALL_PREFIX)/bin
	install tools/zeus-calibrator/zeus-calibrator $(DESTDIR)$(INSTALL_PREFIX)/bin
	install zeus-cli/zeus-cli $(DESTDIR)$(INSTALL_PREFIX)/bin
