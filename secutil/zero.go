package secutil

import "github.com/juthrbog/lazybw/bwcmd"

// ZeroBytes overwrites every element in b with zero. This is effective
// for []byte-backed secrets (e.g. decoded TOTP keys) because Go allows
// in-place mutation of byte slices.
func ZeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// ZeroItem drops references to sensitive string fields on a vault item.
// Go strings are immutable, so we cannot zero their backing memory — we
// can only release references so the GC can collect the underlying data
// sooner and prevent accidental reads of stale secrets.
func ZeroItem(item *bwcmd.Item) {
	item.Notes = ""
	if item.Login != nil {
		item.Login.Password = ""
		item.Login.Totp = ""
	}
	if item.Card != nil {
		item.Card.Number = ""
		item.Card.Code = ""
	}
	if item.Identity != nil {
		item.Identity.SSN = ""
		item.Identity.PassportNumber = ""
		item.Identity.LicenseNumber = ""
	}
	if item.SSHKey != nil {
		item.SSHKey.PrivateKey = ""
	}
}

// ZeroItems calls ZeroItem on every element in the slice.
func ZeroItems(items []bwcmd.Item) {
	for i := range items {
		ZeroItem(&items[i])
	}
}
