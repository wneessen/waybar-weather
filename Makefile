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
