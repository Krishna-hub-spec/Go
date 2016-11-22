// This file' name ends with "_test" to avoid pushing "httptest" import on users.
package gin

import (
	"net/http/httptest"
)

func CreateTestContext() (c *Context, w *httptest.ResponseRecorder, r *Engine) {
	w = httptest.NewRecorder()
	r = New()
	c = r.allocateContext()
	c.reset()
	c.writermem.reset(w)
	return
}
