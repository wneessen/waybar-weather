// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package i18n

import "testing"

func TestNew(t *testing.T) {
	t.Run("new i18n provider with empty locale string succeeds", func(t *testing.T) {
		provider, err := New("")
		if err != nil {
			t.Fatalf("failed to create i18n provider: %s", err)
		}
		if provider == nil {
			t.Fatal("expected i18n provider to be non-nil")
		}
	})
}
