VERSION := $(shell cat ./VERSION)

release:
	git tag -a $(VERSION) -m "release" || true
	git push origin main --tags
.PHONY: release
