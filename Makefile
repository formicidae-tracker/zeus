VERSION := $(shell git describe)
LDFLAGS := -ldflags "-X 'github.com/formicidae-tracker/zeus.ZEUS_VERSION=$(VERSION)'"

all: check-lib zeus/zeus tools/arke-change-device-id/arke-change-device-id tools/arkedump/arkedump tools/arke-zeus-config/arke-zeus-config tools/zeus-calibrator/zeus-calibrator zeus-cli/zeus-cli

check-lib: *.go
	go test && touch check-lib

clean:
	rm -f zeus/zeus
	rm -f tools/arke-change-device-id/arke-change-device-id
	rm -f tools/arkedump/arkedump
	rm -f tools/arke-zeus-config/arke-zeus-config
	rm -f tools/zeus-calibrator/zeus-calibrator
	rm -f zeus-cli/zeus-cli

zeus/zeus: zeus/*.go *.go
	cd zeus && go build $(LDFLAGS) && go test

tools/arke-change-device-id/arke-change-device-id: tools/arke-change-device-id/*.go
	cd tools/arke-change-device-id && go build $(LDFLAGS)

tools/arkedump/arkedump: tools/arkedump/*.go
	cd tools/arkedump && go build $(LDFLAGS)

tools/arke-zeus-config/arke-zeus-config: tools/arke-zeus-config/*.go
	cd tools/arke-zeus-config && go build $(LDFLAGS)

tools/zeus-calibrator/zeus-calibrator: tools/zeus-calibrator/*.go
	cd tools/zeus-calibrator && go build $(LDFLAGS)

zeus-cli/zeus-cli: zeus-cli/*.go *.go
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
