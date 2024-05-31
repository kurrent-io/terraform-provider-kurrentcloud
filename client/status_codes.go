package client

import (
	"io"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

func translateStatusCode(_ int, activity string, body io.Reader) diag.Diagnostics {
	problemDetails, err := newProblemDetailsFromReader(body)
	if err != nil {
		return diag.FromErr(err)
	}

	return diag.Errorf("error %s: %s", activity, problemDetails.Error())
}
