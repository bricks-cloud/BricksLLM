package encryptor

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"google.golang.org/api/idtoken"
)

type Encryptor struct {
	decryptionURL string
	encryptionURL string
	enabled       bool
	client        *http.Client
	timeout       time.Duration
}

type Secret struct {
	Secret string `json:"secret"`
}

type EncryptionResponse struct {
	EncryptedSecret string `json:"encryptedSecret"`
}

type DecryptionResponse struct {
	DecryptedSecret string `json:"decryptedSecret"`
}

func NewEncryptor(decryptionURL string, encryptionURL string, enabled bool, timeout time.Duration, audience string) (Encryptor, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := idtoken.NewClient(ctx, audience)
	if err != nil {
		return Encryptor{}, err
	}

	return Encryptor{
		decryptionURL: decryptionURL,
		encryptionURL: encryptionURL,
		enabled:       enabled,
		timeout:       timeout,
		client:        client,
	}, nil
}

func (e Encryptor) Encrypt(input string, headers map[string]string) (string, error) {
	data, err := json.Marshal(Secret{
		Secret: input,
	})
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.encryptionURL, bytes.NewBuffer(data))
	if err != nil {
		return "", err
	}

	for header, value := range headers {
		req.Header.Add(header, value)
	}

	res, err := e.client.Do(req)
	if err != nil {
		return "", err
	}

	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	encryptionResponse := EncryptionResponse{}
	err = json.Unmarshal(bytes, &encryptionResponse)
	if err != nil {
		return "", err
	}

	return encryptionResponse.EncryptedSecret, nil
}

func (e Encryptor) Enabled() bool {
	return e.enabled && len(e.decryptionURL) != 0 && len(e.encryptionURL) != 0
}

func (e Encryptor) Decrypt(input string, headers map[string]string) (string, error) {
	data, err := json.Marshal(Secret{
		Secret: input,
	})
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.decryptionURL, bytes.NewBuffer(data))
	if err != nil {
		return "", err
	}

	for header, value := range headers {
		req.Header.Add(header, value)
	}

	res, err := e.client.Do(req)
	if err != nil {
		return "", err
	}

	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	decryptionSecret := DecryptionResponse{}
	err = json.Unmarshal(bytes, &decryptionSecret)
	if err != nil {
		return "", err
	}

	return decryptionSecret.DecryptedSecret, nil
}
