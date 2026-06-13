package canvas

import "testing"

// TestClientSatisfiesCanvasAPI asserts at compile time that *Client
// implements the CanvasAPI interface. If methods are added to or
// removed from Client, this test will catch the mismatch.
func TestClientSatisfiesCanvasAPI(t *testing.T) {
	// The compile-time assertion is the real test; this body is a
	// no-op so the test runner counts it.
	var _ CanvasAPI = (*Client)(nil)
}
