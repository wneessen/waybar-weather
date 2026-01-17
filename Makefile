# SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
#
# SPDX-License-Identifier: MIT

.PHONY: i18n
i18n:
	@xspreak -D ./ -p ./internal/i18n/locale/ --copyright-holder 'Winni Neessen <wn@neessen.dev>' --package-name "github.com/wneessen/waybar-weather"

.PHONY: run-ichnaea
run-ichnaea:
	@go build -o tmp/waybar-weather-ichnaea cmd/waybar-weather/main.go
	@sudo setcap cap_net_admin+ep tmp/waybar-weather-ichnaea
	@./tmp/waybar-weather-ichnaea
	@rm tmp/waybar-weather-ichnaea

.PHONY: release
release:
	$(eval TMPDIR := $(shell mktemp -d))
	@go build -o $(TMPDIR)/waybar-weather cmd/waybar-weather/main.go
	@killall waybar-weather 2>/dev/null && true || true
	@sudo cp $(TMPDIR)/waybar-weather /usr/bin/waybar-weather
	@sudo setcap cap_net_admin+ep /usr/bin/waybar-weather
	@rm -rf $(TMPDIR)
