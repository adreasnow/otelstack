package request

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequest(t *testing.T) {
	t.Run("normal case", func(t *testing.T) {
		s := http.Server{Addr: "localhost:45678"}
		go func() {
			mux := http.NewServeMux()
			mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
				w.Write([]byte(`{"key":"value"}`)) //nolint:errcheck
			})

			s.Handler = mux
			if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				t.Logf("error serving server: %v", err)
			}
		}()

		t.Cleanup(func() {
			if err := s.Shutdown(context.Background()); err != nil {
				t.Logf("error shutting down server: %v", err)
			}
		})

		time.Sleep(time.Millisecond + 200)

		var u map[string]string
		err := Request("http://"+s.Addr+"/", &u)
		require.NoError(t, err)

		assert.Contains(t, u, "key")
		assert.Equal(t, "value", u["key"])
	})

	t.Run("no server", func(t *testing.T) {
		var u map[string]string
		err := Request("http://localhost/", &u)
		require.Error(t, err)

		var urlErr *url.Error
		assert.ErrorAs(t, err, &urlErr)
	})

	t.Run("not 200 - retryable", func(t *testing.T) {
		s := http.Server{Addr: "localhost:45679"}
		go func() {
			mux := http.NewServeMux()
			mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusTooManyRequests)
			})

			s.Handler = mux
			if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				t.Logf("error serving server: %v", err)
			}
		}()

		t.Cleanup(func() {
			if err := s.Shutdown(context.Background()); err != nil {
				t.Logf("error shutting down server: %v", err)
			}
		})

		time.Sleep(time.Millisecond + 200)

		var u map[string]string
		err := Request("http://"+s.Addr+"/", &u)
		require.Error(t, err)

		assert.ErrorAs(t, err, &ErrRetryableCode)
	})

	t.Run("not 200 - not retryable", func(t *testing.T) {
		s := http.Server{Addr: "localhost:45680"}
		go func() {
			mux := http.NewServeMux()
			mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusTeapot)
			})

			s.Handler = mux
			if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				t.Logf("error serving server: %v", err)
			}
		}()

		t.Cleanup(func() {
			if err := s.Shutdown(context.Background()); err != nil {
				t.Logf("error shutting down server: %v", err)
			}
		})

		time.Sleep(time.Millisecond + 200)

		var u map[string]string
		err := Request("http://"+s.Addr+"/", &u)
		require.Error(t, err)

		assert.ErrorAs(t, err, &ErrNonRetryableCode)
	})

	t.Run("unmarshalable", func(t *testing.T) {
		s := http.Server{Addr: "localhost:45681"}
		go func() {
			mux := http.NewServeMux()
			mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
				w.Write([]byte(`{"key""value"}`)) //nolint:errcheck
			})

			s.Handler = mux
			if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				t.Logf("error serving server: %v", err)
			}
		}()

		t.Cleanup(func() {
			if err := s.Shutdown(context.Background()); err != nil {
				t.Logf("error shutting down server: %v", err)
			}
		})

		time.Sleep(time.Millisecond + 200)

		var u map[string]string
		err := Request("http://"+s.Addr+"/", &u)
		require.Error(t, err)

		fmt.Printf("%T\n", errors.Unwrap(err))

		var syntaxError *json.SyntaxError
		assert.ErrorAs(t, err, &syntaxError)
	})
}
