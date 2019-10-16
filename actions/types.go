package actions

// Config struct
type Config struct {
	BcName     string
	TargetIP   string
	TargetPort string
}

// XUserAccount xchain user account
type XUserAccount struct {
	Address    string
	PublicKey  []byte
	PrivateKey []byte
}
