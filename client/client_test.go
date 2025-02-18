package client

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"time"

	kitlog "github.com/go-kit/kit/log"
	"github.com/samber/lo"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("New", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	var logger kitlog.Logger

	BeforeEach(func() {
		logger = kitlog.NewLogfmtLogger(kitlog.NewSyncWriter(os.Stderr))
	})

	var (
		server     *httptest.Server
		testClient *ClientWithResponses
		connCount  chan int
	)

	BeforeEach(func() {
		// Channel to track active connections
		connCount = make(chan int, 100)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Simulate something slow that keeps the connection alive
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))

		// Replace the test server's listener with our counting listener
		originalListener := server.Listener
		countingListener := &connectionCountingListener{
			Listener:  originalListener,
			connCount: connCount,
		}
		server.Listener = countingListener

		// Set up the client that we're testing
		apiKey := lo.RandomString(10, lo.AlphanumericCharset)
		c, clientErr := New(ctx, apiKey, server.URL, "1", logger)
		Expect(clientErr).NotTo(HaveOccurred())
		testClient = c
	})

	AfterEach(func() {
		server.Close()
	})

	When("making concurrent requests", func() {
		It("does not exceed the connection count", func() {
			maxConns := 0
			var mu sync.Mutex

			// Start a goroutine to monitor connection counts
			done := make(chan struct{})
			go func() {
				defer close(done)
				for {
					select {
					case count := <-connCount:
						mu.Lock()
						if count > maxConns {
							maxConns = count
						}
						mu.Unlock()
					case <-time.After(2 * time.Second):
						return
					}
				}
			}()

			requestCount := 200

			var wg sync.WaitGroup
			for i := 0; i < requestCount; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					defer GinkgoRecover()

					_, err := testClient.CatalogV2ListTypesWithResponse(ctx)
					Expect(err).NotTo(HaveOccurred())
				}()
			}

			waitDone := make(chan struct{})
			go func() {
				wg.Wait()
				close(waitDone)
			}()

			select {
			case <-waitDone:
			case <-time.After(10 * time.Second):
				Fail("Timed out waiting for requests to complete")
			}

			// Wait for connection counter to finish
			<-done

			mu.Lock()
			defer mu.Unlock()
			Expect(maxConns).To(BeNumerically("<=", 10), "Should not open too many connections")
		})
	})
})

// connectionCountingListener wraps a net.Listener to count active connections
type connectionCountingListener struct {
	net.Listener
	currentConns int
	connMutex    sync.Mutex
	connCount    chan int
}

func (l *connectionCountingListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	l.connMutex.Lock()
	l.currentConns++
	l.connCount <- l.currentConns
	l.connMutex.Unlock()

	return &countingConn{
		Conn:     conn,
		listener: l,
	}, nil
}

// countingConn wraps a net.Conn to track when connections are closed
type countingConn struct {
	net.Conn
	listener *connectionCountingListener
	once     sync.Once
}

func (c *countingConn) Close() error {
	c.once.Do(func() {
		c.listener.connMutex.Lock()
		c.listener.currentConns--
		c.listener.connCount <- c.listener.currentConns
		c.listener.connMutex.Unlock()
	})
	return c.Conn.Close()
}
