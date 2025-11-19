# SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
#
# SPDX-License-Identifier: MIT

.PHONY: i18n
i18n:
	@xspreak -D ./ -p ./internal/i18n/locale/ --copyright-holder 'Winni Neessen <wn@neessen.dev>' --package-name "github.com/wneessen/waybar-weather"
