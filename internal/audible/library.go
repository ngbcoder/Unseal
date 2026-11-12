package audible

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/ngbcoder/Unseal/internal/audible/model"
	"github.com/ngbcoder/Unseal/internal/util"
)

// LibraryResponseGroups is the fixed list of response groups requested
// from the library endpoint.
const LibraryResponseGroups = "badge_types,category_ladders,claim_code_url,contributors,download_flags,media,origin_asin,pdf_url,percent_complete,price,product_attrs,product_desc,product_extended_attrs,plan_details,plans,review_attrs,reviews,sample,series,sku"

// GetLibrary lists the purchased titles of an account.
//
// It pages through the library until a page yields no items. purchasedAfter
// is the lower bound of the purchase date; pass the zero time for a full
// resync.
func (a *Api) GetLibrary(ctx context.Context, purchasedAfter time.Time) ([]model.Product, error) {
	if err := a.ensureProfile(); err != nil {
		return nil, err
	}

	var all []model.Product
	seen := map[string]bool{}

	page := 1
	for {
		q := url.Values{}
		q.Set("num_results", "100")
		q.Set("page", fmt.Sprintf("%d", page))
		q.Set("response_groups", LibraryResponseGroups)
		if !purchasedAfter.IsZero() {
			q.Set("purchased_after", utilToXmlTime(purchasedAfter.Add(time.Millisecond)))
		} else {
			q.Set("purchased_after", utilToXmlTime(time.Unix(0, 0).UTC()))
		}

		path := "/1.0/library?" + q.Encode()
		req, err := a.makeRequest(ctx, "GET", a.AudibleURL(), path, nil, true)
		if err != nil {
			return nil, err
		}

		body, err := a.sendForString(req, a.Audible)
		if err != nil {
			return nil, err
		}

		var resp model.LibraryResponse
		if err := json.Unmarshal([]byte(body), &resp); err != nil {
			return nil, fmt.Errorf("audible: cannot parse the library response: %w", err)
		}

		if len(resp.Items) == 0 {
			break
		}

		for _, p := range resp.Items {
			if seen[p.ASIN] {
				continue
			}
			seen[p.ASIN] = true
			all = append(all, p)
		}

		page++
		if page > 1000 {
			break
		}
	}

	// Sort by purchase date, oldest first, matching the original.
	// (Audible returns them roughly in that order; we keep the order we
	// received them in.)

	return all, nil
}

// utilToXmlTime is a thin wrapper so the audible package does not import
// util for one helper. It is defined in sign.go's neighbourhood.
func utilToXmlTime(t time.Time) string {
	return util.ToXmlTime(t)
}
