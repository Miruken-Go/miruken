package test

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	jwt2 "github.com/golang-jwt/jwt/v5"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/providers/env"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/config"
	koanfp "github.com/miruken-go/miruken/config/koanf"
	"github.com/miruken-go/miruken/provides"
	"github.com/miruken-go/miruken/security/jwt"
	"github.com/miruken-go/miruken/security/login"
	"github.com/miruken-go/miruken/security/login/callback"
	"github.com/miruken-go/miruken/security/principal"
	"github.com/stretchr/testify/suite"
	"math/big"
	"os"
	"strings"
	"testing"
)

type (
	TokenCallbackHandler struct {
		token string
	}
)


// TokenCallbackHandler

func (h *TokenCallbackHandler) Handle(
	c        any,
	greedy   bool,
	composer miruken.Handler,
) miruken.HandleResult {
	switch cb := c.(type) {
	case *callback.Name:
		cb.SetName(h.token)
	default:
		return miruken.NotHandled
	}
	return miruken.Handled
}


type LoginTestSuite struct {
	suite.Suite
	specs []any
}

func (suite *LoginTestSuite) SetupTest() {
	suite.specs = []any{
		&jwt.LoginModule{},
	}
}

func (suite *LoginTestSuite) Setup(specs ...any) (miruken.Handler, error) {
	if len(specs) == 0 {
		specs = suite.specs
	}
	return miruken.Setup().Specs(specs...).Handler()
}

func (suite *LoginTestSuite) TestLogin() {
	suite.Run("Login", func() {
		suite.Run("Succeed", func() {
			handler, _ := suite.Setup()
			ctx := login.New(login.ModuleEntry{Key: "jwt", Options: map[string]any{
				"debug": true,
			}})
			ch := &TokenCallbackHandler{"eyJhbGciOiJSUzI1NiIsImtpZCI6Ilg1ZVhrNHh5b2pORnVtMWtsMll0djhkbE5QNC1jNTdkTzZRR1RWQndhTmsiLCJ0eXAiOiJKV1QifQ.eyJvaWQiOiI4ZTExNzQ1Yi1jZDdlLTQzZTctOTRjNS02MzdmMzJhZGU5NDgiLCJzdWIiOiI4ZTExNzQ1Yi1jZDdlLTQzZTctOTRjNS02MzdmMzJhZGU5NDgiLCJnaXZlbl9uYW1lIjoiQ3JhaWciLCJmYW1pbHlfbmFtZSI6Ik5ldXdpcnQiLCJlbWFpbHMiOlsiY25ldXdpcnRAZ21haWwuY29tIl0sInRmcCI6IkIyQ18xX3NpZ25pbiIsInNjcCI6IlRlYW0uQ3JlYXRlIFBlcnNvbi5DcmVhdGUiLCJhenAiOiJhZWIwMTBjNi1iZjNmLTQxMTMtOGVmZi05NDRjNDg3MmVhNmEiLCJ2ZXIiOiIxLjAiLCJpYXQiOjE2ODc1NTIzMTcsImF1ZCI6IjYwZjEyM2FiLWRlNGQtNGQyZi1iYjkzLWI1NGZkZGMzOGVlMSIsImV4cCI6MTY4NzU1NTkxNywiaXNzIjoiaHR0cHM6Ly90ZWFtc3J2ZGV2Y3JhaWcuYjJjbG9naW4uY29tLzA0OGNmMjA4LTc3OGYtNDk2Yi1iODkyLTlkMDNkMTU2NTJjZC92Mi4wLyIsIm5iZiI6MTY4NzU1MjMxN30.ZxR1ZflO5GDyhbVLDzg67k2v-rb_ns9jbVYhkFLxsaPimUyOrtqMMCPP9vRpQ70YJQQrSo8BaW5Qkorf6wYHjqTAKegA8ucLopWOdmiklRBoPQKdBCQhC7k6S4Kin7YhGwenJiVhJtcL4BLOV8GLXiT_mTj6tpMak6JrzPaJWf-yTW0CqsaLLgoowyCFD4AJEFe0ri_z-uGaZ99zlHWYWkEzcijQt1iEeQRZQbnE-eFPYafrBRzEwMUI1WVf50p7NpVnV79wLlvor7adEJzFEeOP_udXdo6fPKK2QPGDE0bagEzzleOYFoREqlhP61-BW8-iY2t57Y9tenhOBWdQAA"}
			ps := ctx.Login(miruken.AddHandlers(handler, ch))
			suite.NotNil(ps)
			sub, err := ps.Await()
			suite.Nil(err)
			suite.NotNil(sub)
		})

		suite.Run("Fail", func() {
			handler, _ := suite.Setup()
			ctx := login.New(login.ModuleEntry{Key: "jwt"})
			ch  := &TokenCallbackHandler{"eyJhbGciOiJSUzI1NiIsImtpZCI6Ilg1ZVhrNHh5b2pORnVtMWtsMll0djhkbE5QNC1jNTdkTzZRR1RWQndhTmsiLCJ0eXAiOiJKV1QifQ.eyJvaWQiOiI4ZTExNzQ1Yi1jZDdlLTQzZTctOTRjNS02MzdmMzJhZGU5NDgiLCJzdWIiOiI4ZTExNzQ1Yi1jZDdlLTQzZTctOTRjNS02MzdmMzJhZGU5NDgiLCJnaXZlbl9uYW1lIjoiQ3JhaWciLCJmYW1pbHlfbmFtZSI6Ik5ldXdpcnQiLCJlbWFpbHMiOlsiY25ldXdpcnRAZ21haWwuY29tIl0sInRmcCI6IkIyQ18xX3NpZ25pbiIsInNjcCI6IlRlYW0uQ3JlYXRlIFBlcnNvbi5DcmVhdGUiLCJhenAiOiJhZWIwMTBjNi1iZjNmLTQxMTMtOGVmZi05NDRjNDg3MmVhNmEiLCJ2ZXIiOiIxLjAiLCJpYXQiOjE2ODc1NTIzMTcsImF1ZCI6IjYwZjEyM2FiLWRlNGQtNGQyZi1iYjkzLWI1NGZkZGMzOGVlMSIsImV4cCI6MTY4NzU1NTkxNywiaXNzIjoiaHR0cHM6Ly90ZWFtc3J2ZGV2Y3JhaWcuYjJjbG9naW4uY29tLzA0OGNmMjA4LTc3OGYtNDk2Yi1iODkyLTlkMDNkMTU2NTJjZC92Mi4wLyIsIm5iZiI6MTY4NzU1MjMxN30.ZxR1ZflO5GDyhbVLDzg67k2v-rb_ns9jbVYhkFLxsaPimUyOrtqMMCPP9vRpQ70YJQQrSo8BaW5Qkorf6wYHjqTAKegA8ucLopWOdmiklRBoPQKdBCQhC7k6S4Kin7YhGwenJiVhJtcL4BLOV8GLXiT_mTj6tpMak6JrzPaJWf-yTW0CqsaLLgoowyCFD4AJEFe0ri_z-uGaZ99zlHWYWkEzcijQt1iEeQRZQbnE-eFPYafrBRzEwMUI1WVf50p7NpVnV79wLlvor7adEJzFEeOP_udXdo6fPKK2QPGDE0bagEzzleOYFoREqlhP61-BW8-iY2t57Y9tenhOBWdQAA"}
			ps  := ctx.Login(miruken.AddHandlers(handler, ch))
			suite.NotNil(ps)
			sub, err := ps.Await()
			suite.NotNil(err)
			suite.Nil(sub)
			var le login.Error
			suite.ErrorAs(err, &le)
		})
	})

	suite.Run("Logout", func() {
		suite.Run("Succeed", func() {
			handler, _ := suite.Setup()
			ctx := login.New(login.ModuleEntry{Key: "jwt", Options: map[string]any{
				"debug": true,
			}})
			ch := &TokenCallbackHandler{"eyJhbGciOiJSUzI1NiIsImtpZCI6Ilg1ZVhrNHh5b2pORnVtMWtsMll0djhkbE5QNC1jNTdkTzZRR1RWQndhTmsiLCJ0eXAiOiJKV1QifQ.eyJvaWQiOiI4ZTExNzQ1Yi1jZDdlLTQzZTctOTRjNS02MzdmMzJhZGU5NDgiLCJzdWIiOiI4ZTExNzQ1Yi1jZDdlLTQzZTctOTRjNS02MzdmMzJhZGU5NDgiLCJnaXZlbl9uYW1lIjoiQ3JhaWciLCJmYW1pbHlfbmFtZSI6Ik5ldXdpcnQiLCJlbWFpbHMiOlsiY25ldXdpcnRAZ21haWwuY29tIl0sInRmcCI6IkIyQ18xX3NpZ25pbiIsInNjcCI6IlRlYW0uQ3JlYXRlIFBlcnNvbi5DcmVhdGUiLCJhenAiOiJhZWIwMTBjNi1iZjNmLTQxMTMtOGVmZi05NDRjNDg3MmVhNmEiLCJ2ZXIiOiIxLjAiLCJpYXQiOjE2ODc1NTIzMTcsImF1ZCI6IjYwZjEyM2FiLWRlNGQtNGQyZi1iYjkzLWI1NGZkZGMzOGVlMSIsImV4cCI6MTY4NzU1NTkxNywiaXNzIjoiaHR0cHM6Ly90ZWFtc3J2ZGV2Y3JhaWcuYjJjbG9naW4uY29tLzA0OGNmMjA4LTc3OGYtNDk2Yi1iODkyLTlkMDNkMTU2NTJjZC92Mi4wLyIsIm5iZiI6MTY4NzU1MjMxN30.ZxR1ZflO5GDyhbVLDzg67k2v-rb_ns9jbVYhkFLxsaPimUyOrtqMMCPP9vRpQ70YJQQrSo8BaW5Qkorf6wYHjqTAKegA8ucLopWOdmiklRBoPQKdBCQhC7k6S4Kin7YhGwenJiVhJtcL4BLOV8GLXiT_mTj6tpMak6JrzPaJWf-yTW0CqsaLLgoowyCFD4AJEFe0ri_z-uGaZ99zlHWYWkEzcijQt1iEeQRZQbnE-eFPYafrBRzEwMUI1WVf50p7NpVnV79wLlvor7adEJzFEeOP_udXdo6fPKK2QPGDE0bagEzzleOYFoREqlhP61-BW8-iY2t57Y9tenhOBWdQAA"}
			ps := ctx.Login(miruken.AddHandlers(handler, ch))
			suite.NotNil(ps)
			sub, err := ps.Await()
			suite.Nil(err)
			suite.NotNil(sub)
			suite.True(principal.All(sub, principal.User("test")))

			ps = ctx.Logout(handler)
			suite.NotNil(ps)
			sub, err = ps.Await()
			suite.Nil(err)
			suite.NotNil(sub)
			suite.Empty(sub.Principals())
		})
	})

	suite.Run("Configuration", func() {
		suite.Run("Env", func() {
			var k = koanf.New(".")
			_ = os.Setenv("Login_Flow1_0_Key", "module1")
			_ = os.Setenv("Login_Flow1_0_Options_Debug", "true")
			err := k.Load(env.Provider("Login", "_", nil), nil,
				koanf.WithMergeFunc(koanfp.Merge))
			suite.Nil(err)
			handler, _ := miruken.Setup(config.Feature(koanfp.P(k))).Handler()
			cfg, _, err := provides.Type[login.Configuration](handler, &config.Load{Path: "Login"})
			suite.Nil(err)
			suite.NotNil(cfg)
			suite.Len(cfg, 1)
			suite.NotNil(cfg["Flow1"])
			suite.Len(cfg["Flow1"], 1)
			suite.Equal("module1", cfg["Flow1"][0].Key)
			suite.Equal(map[string]any{
				"Debug": "true",
			}, cfg["Flow1"][0].Options)

			f, _, err := provides.Type[login.Flow](handler, &config.Load{Path: "Login.Flow1"})
			suite.Nil(err)
			suite.NotNil(f)
			suite.Equal("module1", f[0].Key)
			suite.Equal(map[string]any{
				"Debug": "true",
			}, f[0].Options)
		})
	})

	suite.Run("Key", func() {
		// https://baptistout.net/posts/convert-jwks-modulus-exponent-to-pem-or-ssh-public-key/

		jwk := map[string]string{
			"n": "tVKUtcx_n9rt5afY_2WFNvU6PlFMggCatsZ3l4RjKxH0jgdLq6CScb0P3ZGXYbPzXvmmLiWZizpb-h0qup5jznOvOr-Dhw9908584BSgC83YacjWNqEK3urxhyE2jWjwRm2N95WGgb5mzE5XmZIvkvyXnn7X8dvgFPF5QwIngGsDG8LyHuJWlaDhr_EPLMW4wHvH0zZCuRMARIJmmqiMy3VD4ftq4nS5s8vJL0pVSrkuNojtokp84AtkADCDU_BUhrc2sIgfnvZ03koCQRoZmWiHu86SuJZYkDFstVTVSR0hiXudFlfQ2rOhPlpObmku68lXw-7V-P7jwrQRFfQVXw",
			"e": "AQAB",
		}
		// decode the base64 bytes for n
		nb, err := base64.RawURLEncoding.DecodeString(jwk["n"])
		if err != nil {
			fmt.Println(err)
		}

		e := 65537
		// The default exponent is usually 65537, so just compare the
		// base64 for [1,0,1] or [0,1,0,1]
		if jwk["e"] != "AQAB" && jwk["e"] != "AAEAAQ" {
			eb, err := base64.RawURLEncoding.DecodeString(jwk["e"])
			if err != nil {
				fmt.Println(err)
			}
			e = int(big.NewInt(0).SetBytes(eb).Uint64())
		}

		pk := &rsa.PublicKey{
			N: new(big.Int).SetBytes(nb),
			E: e,
		}

		der, err := x509.MarshalPKIXPublicKey(pk)
		if err != nil {
			fmt.Println(err)
		}

		block := &pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: der,
		}

		var out bytes.Buffer
		pem.Encode(&out, block)
		fmt.Println(out.String())
	})

	suite.Run("Sig", func() {
		parsedKey, err := jwt2.ParseRSAPublicKeyFromPEM([]byte("-----BEGIN RSA PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAtVKUtcx/n9rt5afY/2WF\nNvU6PlFMggCatsZ3l4RjKxH0jgdLq6CScb0P3ZGXYbPzXvmmLiWZizpb+h0qup5j\nznOvOr+Dhw9908584BSgC83YacjWNqEK3urxhyE2jWjwRm2N95WGgb5mzE5XmZIv\nkvyXnn7X8dvgFPF5QwIngGsDG8LyHuJWlaDhr/EPLMW4wHvH0zZCuRMARIJmmqiM\ny3VD4ftq4nS5s8vJL0pVSrkuNojtokp84AtkADCDU/BUhrc2sIgfnvZ03koCQRoZ\nmWiHu86SuJZYkDFstVTVSR0hiXudFlfQ2rOhPlpObmku68lXw+7V+P7jwrQRFfQV\nXwIDAQAB\n-----END RSA PUBLIC KEY-----"))
		if err != nil {
			suite.Error(err)
		}
		token := "eyJhbGciOiJSUzI1NiIsImtpZCI6Ilg1ZVhrNHh5b2pORnVtMWtsMll0djhkbE5QNC1jNTdkTzZRR1RWQndhTmsiLCJ0eXAiOiJKV1QifQ.eyJvaWQiOiI4ZTExNzQ1Yi1jZDdlLTQzZTctOTRjNS02MzdmMzJhZGU5NDgiLCJzdWIiOiI4ZTExNzQ1Yi1jZDdlLTQzZTctOTRjNS02MzdmMzJhZGU5NDgiLCJnaXZlbl9uYW1lIjoiQ3JhaWciLCJmYW1pbHlfbmFtZSI6Ik5ldXdpcnQiLCJlbWFpbHMiOlsiY25ldXdpcnRAZ21haWwuY29tIl0sInRmcCI6IkIyQ18xX3NpZ25pbiIsInNjcCI6IlRlYW0uQ3JlYXRlIFBlcnNvbi5DcmVhdGUiLCJhenAiOiJhZWIwMTBjNi1iZjNmLTQxMTMtOGVmZi05NDRjNDg3MmVhNmEiLCJ2ZXIiOiIxLjAiLCJpYXQiOjE2ODc1NTIzMTcsImF1ZCI6IjYwZjEyM2FiLWRlNGQtNGQyZi1iYjkzLWI1NGZkZGMzOGVlMSIsImV4cCI6MTY4NzU1NTkxNywiaXNzIjoiaHR0cHM6Ly90ZWFtc3J2ZGV2Y3JhaWcuYjJjbG9naW4uY29tLzA0OGNmMjA4LTc3OGYtNDk2Yi1iODkyLTlkMDNkMTU2NTJjZC92Mi4wLyIsIm5iZiI6MTY4NzU1MjMxN30.ZxR1ZflO5GDyhbVLDzg67k2v-rb_ns9jbVYhkFLxsaPimUyOrtqMMCPP9vRpQ70YJQQrSo8BaW5Qkorf6wYHjqTAKegA8ucLopWOdmiklRBoPQKdBCQhC7k6S4Kin7YhGwenJiVhJtcL4BLOV8GLXiT_mTj6tpMak6JrzPaJWf-yTW0CqsaLLgoowyCFD4AJEFe0ri_z-uGaZ99zlHWYWkEzcijQt1iEeQRZQbnE-eFPYafrBRzEwMUI1WVf50p7NpVnV79wLlvor7adEJzFEeOP_udXdo6fPKK2QPGDE0bagEzzleOYFoREqlhP61-BW8-iY2t57Y9tenhOBWdQAA"
		parts := strings.Split(token, ".")
		sig, err := jwt2.NewParser().DecodeSegment(parts[2])
		if err != nil {
			suite.Error(err)
		}
		err = jwt2.SigningMethodRS256.Verify(strings.Join(parts[0:2], "."), sig, parsedKey)
		if err != nil {
			suite.Error(err)
		}
	})
}

func TestLoginTestSuite(t *testing.T) {
	suite.Run(t, new(LoginTestSuite))
}
