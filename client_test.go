package lakefs

func testClient() *Client {
	c := NewClient(testCredential(), &Options{
		Debug: true,
	})
	return c
}
