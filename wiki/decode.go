package wiki

// Flag decodes a Bucket BOOLEAN field. Bucket sends the field, with an
// empty value, when true and omits it when false.
type Flag bool

func (f *Flag) UnmarshalJSON([]byte) error {
	*f = true
	return nil
}
