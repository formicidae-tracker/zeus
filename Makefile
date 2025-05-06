all: cmd check


cmd: zeuspb zeus
	make -C cmd

zeus:
	make -C internal/zeus

zeuspb:
	make -C pkg/zeuspb

check: zeuspb zeus
	make -C internal/zeus check
	make -C pkg/zeuspb check
	make -C cmd check


clean:
	make -C cmd clean

INSTALL_PREFIX=/usr/local

.PHONY: clean check cmd zeuspb zeus

install: all
	mkdir -p $(DESTDIR)$(INSTALL_PREFIX)/bin
	install cmd/zeus/zeus $(DESTDIR)$(INSTALL_PREFIX)/bin
	install cmd/arke-change-device-id/arke-change-device-id $(DESTDIR)$(INSTALL_PREFIX)/bin
	install cmd/arke-zeus-config/arke-zeus-config $(DESTDIR)$(INSTALL_PREFIX)/bin
	install cmd/zeus-calibrator/zeus-calibrator $(DESTDIR)$(INSTALL_PREFIX)/bin
	install cmd/zeus-cli/zeus-cli $(DESTDIR)$(INSTALL_PREFIX)/bin
