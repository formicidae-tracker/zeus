all: zeus zeus/zeus zeus-cli/zeus-cli tools check

.PHONY: zeus clean check tools

zeus: *.go zeuspb/*.go zeuspb/*.proto
	go generate
	go build

tools:
	make -C tools

check:
	go test
	go vet
	make -C zeus check
	make -C zeus-cli check
	make -C tools check

zeus/zeus: *.go zeus/*.go zeuspb/*.go zeuspb/*.proto
	make -C zeus

zeus-cli/zeus-cli: *.go zeus-cli/*.go zeuspb/*.go zeuspb/*.proto
	make -C zeus-cli

clean:
	make -C zeus clean
	make -C zeus-cli clean
	make -C tools clean

INSTALL_PREFIX=/usr/local

install: all
	mkdir -p $(DESTDIR)$(INSTALL_PREFIX)/bin
	install zeus/zeus $(DESTDIR)$(INSTALL_PREFIX)/bin
	install tools/arke-change-device-id/arke-change-device-id $(DESTDIR)$(INSTALL_PREFIX)/bin
	install tools/arkedump/arkedump $(DESTDIR)$(INSTALL_PREFIX)/bin
	install tools/arke-zeus-config/arke-zeus-config $(DESTDIR)$(INSTALL_PREFIX)/bin
	install tools/zeus-calibrator/zeus-calibrator $(DESTDIR)$(INSTALL_PREFIX)/bin
	install zeus-cli/zeus-cli $(DESTDIR)$(INSTALL_PREFIX)/bin
