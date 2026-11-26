package utils

// Seal encrypts a secret (an API token) with AES-256-GCM for storage.
func Seal(cfg AppConfig, secret string) (string, error) {
	return encrypt(cfg.EncryptionKey, []byte(secret))
}

// Open reverses Seal.
func Open(cfg AppConfig, sealed string) (string, error) {
	raw, err := decrypt(cfg.EncryptionKey, sealed)
	return string(raw), err
}
