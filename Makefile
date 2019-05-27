all: lib zeus/zeus arke-change-device-id arkedump arke-zeus-config zeus-calibrator

lib:
	go test

zeus/zeus:
	cd zeus && go build && go test

arke-change-device-id:
	cd tools/arke-change-device-id && go build

arkedump:
	cd tools/arkedump && go build

arke-zeus-config:
	cd tools/arke-zeus-config && go build

zeus-calibrator:
	cd tools/zeus-calibrator && go build

INSTALL_PREFIX=/usr/local

install: all
	mkdir -p $(DESTDIR)$(INSTALL_PREFIX)/bin
	install zeus/zeus $(DESTDIR)$(INSTALL_PREFIX)/bin
	install tools/arke-change-device-id/arke-change-device-id $(DESTDIR)$(INSTALL_PREFIX)/bin
	install tools/arkedump/arkedump $(DESTDIR)$(INSTALL_PREFIX)/bin
	install tools/arke-zeus-config/arke-zeus-config $(DESTDIR)$(INSTALL_PREFIX)/bin
	install tools/zeus-calibrator/zeus-calibrator $(DESTDIR)$(INSTALL_PREFIX)/bin
