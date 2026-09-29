package planetscale

type ExternalDataSourceSSLVerificationMode int

const (
	SSLModeDisabled ExternalDataSourceSSLVerificationMode = iota
	SSLModePreferred
	SSLModeRequired
	SSLModeVerifyCA
	SSLModeVerifyIdentity
)

func (sm ExternalDataSourceSSLVerificationMode) String() string {
	switch sm {
	case SSLModeDisabled:
		return "disabled"
	case SSLModePreferred:
		return "preferred"
	case SSLModeRequired:
		return "required"
	case SSLModeVerifyCA:
		return "verify_ca"
	default:
		return "verify_identity"
	}
}
