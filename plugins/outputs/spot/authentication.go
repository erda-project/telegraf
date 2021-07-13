package spot

import (
	"net/http"

	"github.com/influxdata/telegraf/plugins/common/secret"
	"github.com/influxdata/telegraf/plugins/common/secret/hmac"
)

type Secret interface {
	Secure(req *http.Request)
}

type basicSecret struct {
	username, password string
}

func (b *basicSecret) Secure(req *http.Request) {
	req.SetBasicAuth(b.username, b.password)
}

type hmacSecret struct {
	signer *hmac.Signer
}

func (h *hmacSecret) Secure(req *http.Request) {
	h.signer.SignCanonicalRequest(req)
}

type noSecret struct {
}

func (n *noSecret) Secure(req *http.Request) {
	return
}

type authConfig struct {
	Type     string            `toml:"type"`
	Property map[string]string `toml:"property"`
}

func createSecret(cfg authConfig) Secret {
	switch cfg.Type {
	case "basic":
		return &basicSecret{
			username: cfg.Property["auth_username"],
			password: cfg.Property["auth_password"],
		}
	case "hmac":
		return &hmacSecret{
			signer: hmac.New(secret.AkSkPair{
				AccessKeyID: cfg.Property["access_key_id"],
				SecretKey:   cfg.Property["secret_key"],
			}),
		}
	default:
		return &noSecret{}
	}
}
