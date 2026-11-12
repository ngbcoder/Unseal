package audible

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ngbcoder/Unseal/internal/audible/model"
)

// GetProductInfo fetches the catalogue record of one title.
func (a *Api) GetProductInfo(ctx context.Context, asin string) (*model.Product, error) {
	if err := a.ensureProfile(); err != nil {
		return nil, err
	}

	path := "/1.0/catalog/products/" + asin + "?response_groups=contributors,media,product_attrs,product_desc,product_extended_attrs"
	req, err := a.makeRequest(ctx, "GET", a.AmazonURL(), path, nil, true)
	if err != nil {
		return nil, err
	}

	body, err := a.sendForString(req, a.Amazon)
	if err != nil {
		return nil, err
	}

	var resp model.ProductResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return nil, fmt.Errorf("audible: cannot parse the product response: %w", err)
	}
	return &resp.Product, nil
}
