all: arke-change-device-id/arke-change-device-id arkedump/arkedump arke-zeus-config/arke-zeus-config zeus-calibrator/zeus-calibrator check

.PHONY: clean check

arke-change-device-id/arke-change-device-id: arke-change-device-id/*.go
	make -C arke-change-device-id

arkedump/arkedump: arkedump/*.go
	make -C arkedump

arke-zeus-config/arke-zeus-config: arke-zeus-config/*.go
	make -C arke-zeus-config

zeus-calibrator/zeus-calibrator: zeus-calibrator/*.go
	make -C zeus-calibrator

clean:
	make -C arke-change-device-id clean
	make -C arkedump clean
	make -C arke-zeus-config clean
	make -C zeus-calibrator clean

check:
	make -C arke-change-device-id check
	make -C arkedump check
	make -C arke-zeus-config check
	make -C zeus-calibrator check
