package e2e_test

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/input"
	cdpruntime "github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type datasourceInfo struct {
	name       string
	id         int
	uid        string
	uiSelector string
}

var _ = Describe("The plugin", func() {

	datasourceTable := `div[class$="-page-content"] ul`
	preprovisionedAlert := `div[data-testid="data-testid Alert info"]`
	datasources := []datasourceInfo{
		{
			name:       "mongodb",
			id:         1,
			uid:        "P1CC9A79BDAF09793",
			uiSelector: datasourceTable + ` > li:nth-child(1) > div > h2 > a`,
		},
		{
			name:       "mongodb-mlts",
			id:         4,
			uid:        "P2CE72B00BA39C90B",
			uiSelector: datasourceTable + ` > li:nth-child(2) > div > h2 > a`,
		},
		{
			name:       "mongodb-no-auth",
			id:         3,
			uid:        "P37BB7E38C06C68DF",
			uiSelector: datasourceTable + ` > li:nth-child(3) > div > h2 > a`,
		},
		{
			name:       "mongodb-non-default-auth-source",
			id:         2,
			uid:        "P96F3DFDC70C53703",
			uiSelector: datasourceTable + ` > li:nth-child(4) > div > h2 > a`,
		},
		{
			name:       "mongodb-tls",
			id:         5,
			uid:        "P5C7DC0BAB25D7937",
			uiSelector: datasourceTable + ` > li:nth-child(5) > div > h2 > a`,
		},
		{
			name:       "mongodb-tls-insecure",
			id:         6,
			uid:        "P50072D096AA1FAA5",
			uiSelector: datasourceTable + ` > li:nth-child(5) > div > h2 > a`,
		},
	}

	for _, datasource := range datasources {
		datasource := datasource
		Describe("on the ui", func() {
			It(fmt.Sprintf("should load a pre-provisioned %s datasource on the UI", datasource.name), func() {
				b.Prepare()

				b.Navigate("http://grafana.grafana-mongodb-it.cluster/datasources")
				Eventually(datasource.uiSelector, "15s").Should(b.Exist())
				b.Click(datasource.uiSelector)
				Eventually(preprovisionedAlert, "15s").Should(b.Exist())
			})
		})

		for _, query := range queries {
			query := query
			It(fmt.Sprintf("should execute the %s query against the %s datasource", query.Name, datasource.name), func() {
				datasource := datasource

				for ix := range query.Body.Queries {
					query.Body.Queries[ix].DatasourceID = datasource.id
					query.Body.Queries[ix].Datasource.UID = datasource.uid
				}
				queryBytes, err := io.ReadAll(query.Reader())
				Expect(err).ToNot(HaveOccurred())
				GinkgoWriter.Printf("Executing query %s\n", string(queryBytes))

				req, err := http.NewRequest(http.MethodPost, "http://grafana.grafana-mongodb-it.cluster/api/ds/query", query.Reader())
				Expect(err).ToNot(HaveOccurred())
				req.SetBasicAuth("admin", "adminPassword")
				req.Header.Set("accept", "application/json, text/plain, */*")
				req.Header.Set("content-type", "application/json")
				resp, err := clusterHTTPClient.Do(req)
				Expect(err).ToNot(HaveOccurred())
				GinkgoWriter.Print("Got response ")
				_, err = io.Copy(GinkgoWriter, resp.Body)
				Expect(err).ToNot(HaveOccurred())
				GinkgoWriter.Println("")
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
			})
		}

		createFolder := "weather/alerts/create-folder"
		createRuleGroup := "weather/alerts/create-rule-group"
		It(fmt.Sprintf("should execute the %s query", createFolder), func() {
			f, err := os.Open(filepath.Join("../integration-test/queries", createFolder+".json"))
			Expect(err).ToNot(HaveOccurred())
	
			req, err := http.NewRequest(http.MethodPost, "http://grafana.grafana-mongodb-it.cluster/api/folders", f)
			Expect(err).ToNot(HaveOccurred())
			req.SetBasicAuth("admin", "adminPassword")
			req.Header.Set("accept", "application/json, text/plain, */*")
			req.Header.Set("content-type", "application/json")
			resp, err := client.Do(req)
			Expect(err).ToNot(HaveOccurred())
			_, err = io.Copy(GinkgoWriter, resp.Body)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})
		It(fmt.Sprintf("should execute the %s query", createRuleGroup), func() {
			f, err := os.Open(filepath.Join("../integration-test/queries", createFolder+".json"))
			Expect(err).ToNot(HaveOccurred())
	
			req, err := http.NewRequest(http.MethodPost, "http://grafana.grafana-mongodb-it.cluster/api/ruler/grafana/api/v1/rules/alert_folder?subtype=cortex", f)
			Expect(err).ToNot(HaveOccurred())
			req.SetBasicAuth("admin", "adminPassword")
			req.Header.Set("accept", "application/json, text/plain, */*")
			req.Header.Set("content-type", "application/json")
			resp, err := client.Do(req)
			Expect(err).ToNot(HaveOccurred())
			_, err = io.Copy(GinkgoWriter, resp.Body)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})
		It(fmt.Sprintf("should execute the alerts evaluation query", createRuleGroup), func() {
			req, err := http.NewRequest(http.MethodGet, "http://grafana.grafana-mongodb-it.cluster/api/prometheus/grafana/api/v1/rules", nil)
			Expect(err).ToNot(HaveOccurred())
			req.SetBasicAuth("admin", "adminPassword")
			req.Header.Set("accept", "application/json, text/plain, */*")
			resp, err := client.Do(req)
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
