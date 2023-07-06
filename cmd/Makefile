all: zeus/zeus zeus-cli/zeus-cli arke-change-device-id/arke-change-device-id arkedump/arkedump arke-zeus-config/arke-zeus-config zeus-calibrator/zeus-calibrator check

.PHONY: clean check

zeus/zeus:
	make -C zeus

zeus-cli/zeus-cli:
	make -C zeus-cli

arke-change-device-id/arke-change-device-id:
	make -C arke-change-device-id

arkedump/arkedump:
	make -C arkedump

arke-zeus-config/arke-zeus-config:
	make -C arke-zeus-config

zeus-calibrator/zeus-calibrator:
	make -C zeus-calibrator

clean:
	make -C zeus clean
	make -C zeus-cli clean
	make -C arke-change-device-id clean
	make -C arkedump clean
	make -C arke-zeus-config clean
	make -C zeus-calibrator clean

check:
	make -C zeus check
	make -C zeus-cli check
	make -C arke-change-device-id check
	make -C arkedump check
	make -C arke-zeus-config check
	make -C zeus-calibrator check
