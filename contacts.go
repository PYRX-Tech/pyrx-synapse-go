package synapse

import "strconv"

// ContactsClient provides methods for managing contacts.
type ContactsClient struct {
	client *Client
}

// List returns a paginated list of contacts.
func (cc *ContactsClient) List(params ContactListParams) (*ContactListResponse, error) {
	qp := map[string]string{}
	if params.Search != "" {
		qp["search"] = params.Search
	}
	if params.Page > 0 {
		qp["page"] = strconv.Itoa(params.Page)
	}
	if params.PerPage > 0 {
		qp["per_page"] = strconv.Itoa(params.PerPage)
	}
	if params.SortBy != "" {
		qp["sort_by"] = params.SortBy
	}
	if params.SortOrder != "" {
		qp["sort_order"] = params.SortOrder
	}

	var resp ContactListResponse
	if err := cc.client.requestWithParams("GET", "/v1/contacts", nil, qp, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Get retrieves a single contact by ID.
func (cc *ContactsClient) Get(contactID string) (*ContactResponse, error) {
	var resp ContactResponse
	if err := cc.client.request("GET", "/v1/contacts/"+contactID, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Update updates a contact by external ID.
func (cc *ContactsClient) Update(externalID string, params ContactUpdateParams) (*ContactResponse, error) {
	var resp ContactResponse
	if err := cc.client.requestWithParams("PATCH", "/v1/contacts/"+externalID, params, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Delete removes a contact by external ID.
func (cc *ContactsClient) Delete(externalID string) error {
	return cc.client.request("DELETE", "/v1/contacts/"+externalID, nil, nil)
}
