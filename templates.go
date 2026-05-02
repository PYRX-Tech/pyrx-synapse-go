package synapse

// TemplatesClient provides methods for managing email templates.
type TemplatesClient struct {
	client *Client
}

// List returns all email templates.
func (tc *TemplatesClient) List() ([]TemplateResponse, error) {
	var resp []TemplateResponse
	if err := tc.client.request("GET", "/v1/templates", nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// Get retrieves a single template by slug.
func (tc *TemplatesClient) Get(slug string) (*TemplateResponse, error) {
	var resp TemplateResponse
	if err := tc.client.request("GET", "/v1/templates/"+slug, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Create creates a new email template.
func (tc *TemplatesClient) Create(params TemplateCreateParams) (*TemplateResponse, error) {
	var resp TemplateResponse
	if err := tc.client.request("POST", "/v1/templates", params, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Update updates an existing template by slug.
func (tc *TemplatesClient) Update(slug string, params TemplateUpdateParams) (*TemplateResponse, error) {
	var resp TemplateResponse
	if err := tc.client.request("PUT", "/v1/templates/"+slug, params, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Delete removes a template by slug.
func (tc *TemplatesClient) Delete(slug string) error {
	return tc.client.request("DELETE", "/v1/templates/"+slug, nil, nil)
}

// Preview renders a template with sample data.
func (tc *TemplatesClient) Preview(slug string, params TemplatePreviewParams) (*TemplatePreviewResponse, error) {
	var resp TemplatePreviewResponse
	if err := tc.client.request("POST", "/v1/templates/"+slug+"/preview", params, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
