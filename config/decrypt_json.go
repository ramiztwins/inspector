package config

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"encoding/base64"
	"errors"
	"os"
	"inspector/mylogger"
)

// Read and parse the RSA private key from a PEM file.
// Returns the RSA private key or an error if the key cannot be loaded.
func loadPrivateKey() (*rsa.PrivateKey, error) {
	const privateKeyPath = "/run/secrets/private_key"
	privateKeyData, err := os.ReadFile(privateKeyPath)  
	if err != nil {
		mylogger.MainLogger.Errorf("Failed to read private key at path: %s with error: %s", privateKeyPath, err)
		return nil, err
	}

	block, _ := pem.Decode(privateKeyData)
	if block == nil || block.Type != "PRIVATE KEY" {
		mylogger.MainLogger.Errorf("Invalid PEM block when decoding private key at path: %s", privateKeyPath)
		return nil, errors.New("invalid PEM block")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		mylogger.MainLogger.Errorf("Failed to parse private key at path: %s with error: %s", privateKeyPath, err)
		return nil, err
	}

	if rsaKey, ok := key.(*rsa.PrivateKey); ok {
		return rsaKey, nil
	}

	mylogger.MainLogger.Errorf("Not an RSA private key at path: %s", privateKeyPath)
	return nil, errors.New("not an RSA private key")
}

// decryptRSA decrypts a base64-encoded ciphertext using the provided RSA private key.
// Returns the decrypted string or an error if decryption fails.
func decryptRSA(ciphertext string, privKey *rsa.PrivateKey) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		mylogger.MainLogger.Errorf("Failed to decode ciphertext: %s", err)
		return "", err
	}

	decrypted, err := rsa.DecryptPKCS1v15(nil, privKey, data)
	if err != nil {
		mylogger.MainLogger.Errorf("Failed to decrypt data with RSA: %s", err)
		return "", err
	}

	return string(decrypted), nil
}

// DecryptConfig decrypts all encrypted cookie values in the provided config object.
// Returns an error if any decryption operation fails.
func DecryptConfig(config *Config) error {
	privKey, err := loadPrivateKey()
	if err != nil {
		mylogger.MainLogger.Errorf("Failed to load private key: %s", err)
		return err
	}

	for _, target := range config.Targets {
		for _, prober := range target.Probers {
			for cookieName, encryptedValue := range prober.Context.Cookies {
				decryptedValue, err := decryptRSA(encryptedValue, privKey)
				if err != nil {
					mylogger.MainLogger.Errorf("Failed to decrypt cookie value for target %s, prober %s, cookie %s: %s", target.Id, prober.Id, cookieName, err)
					return err
				}
                
				mylogger.MainLogger.Infof("Decrypted cookie '%s' for target '%s', prober '%s'. Decrypted value: %s", cookieName, target.Id, prober.Id, decryptedValue)
				prober.Context.Cookies[cookieName] = decryptedValue
			}
		}
	}

	mylogger.MainLogger.Infof("Successfully decrypted cookies for all targets")
	return nil
}