// Package size contains definitions of common units of data.
package size

// A Size represents a size of a portion of data expressed in bytes.
type Size int64

// Common units of data.
const (
	Byte Size = 1

	Kilobyte = 1000 * Byte
	Megabyte = 1000 * Kilobyte
	Gigabyte = 1000 * Megabyte
	Terabyte = 1000 * Gigabyte
	Petabyte = 1000 * Terabyte
	Exabyte  = 1000 * Petabyte

	Kibibyte = 1024 * Byte
	Mebibyte = 1024 * Kibibyte
	Gibibyte = 1024 * Mebibyte
	Tebibyte = 1024 * Gibibyte
	Pebibyte = 1024 * Tebibyte
	Exbibyte = 1024 * Pebibyte
)
