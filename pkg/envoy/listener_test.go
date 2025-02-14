package envoy

import (
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	v1 "k8s.io/api/core/v1"
	"os"
	"testing"
)

func TestCreateHTTPListener(t *testing.T) {
	manager := httpConnectionManager([]*route.VirtualHost{})
	KubeClient := newMockedKubeClientListener("", "")

	l, err := newExternalEnvoyListener(false, &manager, KubeClient)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, core.TCP, l.Address.GetSocketAddress().Protocol)
	assert.Equal(t, "0.0.0.0", l.Address.GetSocketAddress().Address)
	assert.Equal(t, uint32(8080), l.Address.GetSocketAddress().GetPortValue())
	assert.Assert(t, is.Nil(l.FilterChains[0].TlsContext)) //TLS not configured
}

func TestCreateHTTPSListener(t *testing.T) {
	err := setHTTPSEnvs()
	if err != nil {
		t.Error(err)
	}

	cert := "some_cert"
	key := "some_key"
	KubeClient := newMockedKubeClientListener(cert, key)

	manager := httpConnectionManager([]*route.VirtualHost{})

	l, err := newExternalEnvoyListener(true, &manager, KubeClient)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, core.TCP, l.Address.GetSocketAddress().Protocol)
	assert.Equal(t, "0.0.0.0", l.Address.GetSocketAddress().Address)
	assert.Equal(t, uint32(8443), l.Address.GetSocketAddress().GetPortValue())

	// Check that TLS is configured
	certs := l.FilterChains[0].TlsContext.CommonTlsContext.TlsCertificates[0]
	assert.Equal(t, cert, string(certs.CertificateChain.GetInlineBytes()))
	assert.Equal(t, key, string(certs.PrivateKey.GetInlineBytes()))
}

type mockedKubeClientListener struct {
	cert string
	key  string
}

func (kubeClient *mockedKubeClientListener) EndpointsForRevision(
	namespace string, serviceName string) (*v1.EndpointsList, error) {

	return nil, nil
}

func (kubeClient *mockedKubeClientListener) ServiceForRevision(
	namespace string, serviceName string) (*v1.Service, error) {

	return nil, nil
}

func (kubeClient *mockedKubeClientListener) GetSecret(
	namespace string, secretName string) (*v1.Secret, error) {

	secret := v1.Secret{
		Data: map[string][]byte{
			"tls.crt": []byte(kubeClient.cert),
			"tls.key": []byte(kubeClient.key),
		},
	}

	return &secret, nil
}

func newMockedKubeClientListener(cert string, key string) *mockedKubeClientListener {
	return &mockedKubeClientListener{cert: cert, key: key}
}

func setHTTPSEnvs() error {
	err := os.Setenv("CERTS_SECRET_NAMESPACE", "default")
	if err != nil {
		return err
	}

	err = os.Setenv("CERTS_SECRET_NAME", "kourier-certs")
	if err != nil {
		return err
	}

	return nil
}
