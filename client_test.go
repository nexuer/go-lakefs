package lakefs

func testClient(nodebug ...bool) *Client {
	debug := true
	if len(nodebug) > 0 && nodebug[0] {
		debug = false
	}
	c := NewClient(testCredential(), &Options{
		Debug: debug,
	})
	return c
}
