package e2e_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/input"
	cdpruntime "github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("The plugin", func() {

	datasourceTable := `div[class$="-page-content"] ul`
	mongodbDatasource := datasourceTable + ` > li:nth-child(1) > div > h2 > a`
	mongodbMTLSDatasource := datasourceTable + ` > li:nth-child(2) > div > h2 > a`
	mongodbNoAuthDatasource := datasourceTable + ` > li:nth-child(3) > div > h2 > a`
	mongodbTLSDatasource := datasourceTable + ` > li:nth-child(4) > div > h2 > a`
	mongodbTLSInsecureDatasource := datasourceTable + ` > li:nth-child(5) > div > h2 > a`

	preprovisionedAlert := `div[data-testid="data-testid Alert info"]`

	datasourceTests := map[string]string{
		"plaintext":         mongodbDatasource,
		"mTLS":              mongodbMTLSDatasource,
		"no authentication": mongodbNoAuthDatasource,
		"TLS":               mongodbTLSDatasource,
		"insecure TLS":      mongodbTLSInsecureDatasource,
	}

	for desc, datasource := range datasourceTests {
		It(fmt.Sprintf("should load a pre-provisioned %s datasource", desc), func() {

			b.Prepare()

			b.Navigate("http://grafana.grafana-mongodb-it.cluster/datasources")
			Eventually(datasource, "15s").Should(b.Exist())
			b.Click(datasource)
			Eventually(preprovisionedAlert, "15s").Should(b.Exist())
		})
	}

	queries := []string{
		"weather/timeseries",
		"weather/timeseries-date",
		"weather/table",
		"tweets/timeseries",
		"conversion_check/table",
	}

	for _, query := range queries {
		It(fmt.Sprintf("should execute the %s query", query), func() {
			f, err := os.Open(filepath.Join("../integration-test/queries", query+".json"))
			Expect(err).ToNot(HaveOccurred())

			req, err := http.NewRequest(http.MethodPost, "http://grafana.grafana-mongodb-it.cluster/api/ds/query", f)
			Expect(err).ToNot(HaveOccurred())
			req.SetBasicAuth("admin", "adminPassword")
			req.Header.Set("accept", "application/json, text/plain, */*")
			req.Header.Set("content-type", "application/json")
			resp, err := clusterHTTPClient.Do(req)
			Expect(err).ToNot(HaveOccurred())
			_, err = io.Copy(GinkgoWriter, resp.Body)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})
	}

})

func nodes(sel interface{}) []*cdp.Node {
	GinkgoHelper()
	var toReturn []*cdp.Node
	Expect(chromedp.Run(b.Context, chromedp.QueryAfter(sel, func(ctx context.Context, id cdpruntime.ExecutionContextID, nodes ...*cdp.Node) error {
		toReturn = nodes
		return nil
	}))).To(Succeed())
	return toReturn
}

func mouseClick(sel interface{}, opts ...chromedp.MouseOption) {
	GinkgoHelper()
	Expect(chromedp.Run(b.Context, chromedp.QueryAfter(sel, func(ctx context.Context, id cdpruntime.ExecutionContextID, nodes ...*cdp.Node) error {
		Expect(nodes).To(HaveLen(1))
		return chromedp.MouseClickNode(nodes[0], opts...).Do(ctx)
	}))).To(Succeed())
}

func mouseMove(sel interface{}, opts ...chromedp.MouseOption) {
	GinkgoHelper()
	Expect(chromedp.Run(b.Context, chromedp.QueryAfter(sel, func(ctx context.Context, id cdpruntime.ExecutionContextID, nodes ...*cdp.Node) error {
		Expect(nodes).To(HaveLen(1))
		return MouseMoveNode(nodes[0], opts...).Do(ctx)
	}))).To(Succeed())
}

// MouseClickXY is an action that sends a left mouse button click (i.e.,
// mousePressed and mouseReleased event) to the X, Y location.
func MouseMoveXY(x, y float64, opts ...chromedp.MouseOption) chromedp.MouseAction {
	GinkgoHelper()
	return chromedp.ActionFunc(func(ctx context.Context) error {
		p := &input.DispatchMouseEventParams{
			Type: input.MouseMoved,
			X:    x,
			Y:    y,
		}

		// apply opts
		for _, o := range opts {
			p = o(p)
		}

		return p.Do(ctx)
	})
}

// MouseClickNode is an action that dispatches a mouse left button click event
// at the center of a specified node.
//
// Note that the window will be scrolled if the node is not within the window's
// viewport.
func MouseMoveNode(n *cdp.Node, opts ...chromedp.MouseOption) chromedp.MouseAction {
	GinkgoHelper()
	return chromedp.ActionFunc(func(ctx context.Context) error {
		t := cdp.ExecutorFromContext(ctx).(*chromedp.Target)
		if t == nil {
			return chromedp.ErrInvalidTarget
		}

		if err := dom.ScrollIntoViewIfNeeded().WithNodeID(n.NodeID).Do(ctx); err != nil {
			return err
		}

		boxes, err := dom.GetContentQuads().WithNodeID(n.NodeID).Do(ctx)
		if err != nil {
			return err
		}

		if len(boxes) == 0 {
			return chromedp.ErrInvalidDimensions
		}

		content := boxes[0]

		c := len(content)
		if c%2 != 0 || c < 1 {
			return chromedp.ErrInvalidDimensions
		}

		var x, y float64
		for i := 0; i < c; i += 2 {
			x += content[i]
			y += content[i+1]
		}
		x /= float64(c / 2)
		y /= float64(c / 2)

		return MouseMoveXY(x, y, opts...).Do(ctx)
	})
}
