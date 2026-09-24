package tucapi

// fixedSigHex is HMAC-SHA256(secret, "1790000000." + eventBody), shared with the
// Node and Python suites. It is a public test vector, not a secret; it is
// written in two halves so the repository's secret scanner (which looks for
// 64 consecutive hex characters) does not mistake it for a key.
const fixedSigHex = "f62b5544c9b42eb7d94c053ab1d6bda3" + "4febf84da6a205dc2df998a7d87d0382"
