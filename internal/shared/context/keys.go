package context

type key string

const (
	KeyEndpointSchema key = "gateway.endpoint_schema"
	KeyPIIMasking     key = "gateway.pii_masking"
	KeyEndpointID     key = "gateway.endpoint_id"
	KeyAppID          key = "gateway.app_id"
	KeyOrgID          key = "gateway.org_id"
	KeyUserID         key = "gateway.user_id"
)
