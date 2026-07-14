package toc

// tocRoastTable is the XOR key used to "roast" (obfuscate) TOC passwords. It
// mirrors wire.RoastTOCPassword in the server so that this client produces
// passwords the server accepts. The key is the ASCII string "Tic/Toc".
var tocRoastTable = []byte("Tic/Toc")

// RoastPassword obfuscates a cleartext TOC password by XOR-ing each byte with
// the TOC roast table ("Tic/Toc"). Roasting is symmetric: applying it again
// recovers the original cleartext. Clients send the roasted bytes (hex-encoded
// with an "0x" prefix) in the toc_signon command.
func RoastPassword(clearPassword []byte) []byte {
	roasted := make([]byte, len(clearPassword))
	for i := range clearPassword {
		roasted[i] = clearPassword[i] ^ tocRoastTable[i%len(tocRoastTable)]
	}
	return roasted
}
